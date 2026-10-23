import type { ServerMessage } from '../lobby/data_models';
import { createMessage } from '../lobby/data_models';
import { Sounds } from '../../sounds/Sounds';
import { perfEvent } from '../../scripts/perf';
import type { UpdateMessage } from './data_models';
import type { DwgGame } from './game';

/** After this many repeated errors the frontend should shutdown */
const MAX_ERROR_COUNT = 3;

function isGameMessage(kind: string): boolean {
  const lobby_prefixes = ['game', 'ping'];
  return !lobby_prefixes.every((prefix) => !kind.startsWith(`${prefix}-`));
}

/** Advances the setting revision only for the next one in order, so a missed update stays visible to the heartbeat */
function trackSettingsRev(game: DwgGame, rev: number | undefined): void {
  const game_base = game.getGame()?.game_base;
  if (game_base && rev === (game_base.settings_rev ?? 0) + 1) {
    game_base.settings_rev = rev;
  }
}

/** Handles messages for the frontend lobby */
export function handleMessage(game: DwgGame, message: ServerMessage): void {
  if (!game.socketActive()) {
    return;
  }
  if (!isGameMessage(message.kind)) {
    return;
  }
  let errored = false;
  if (message.kind === 'game-update' && game.getGame()?.game_base.persistant_history === false) {
    console.log(`Game message: ${message.data} id=${message.sender.split('-')[2]} bytes=${message.content.length}`);
  } else if (message.kind !== 'ping-update') {
    console.log('Game message:', message);
  }
  switch (message.kind) {
    case 'ping-update':
      // TODO: implement
      return; // doesn't affect error count
    case 'game-chat':
      const chat_client_id = parseInt((message.sender ?? '').replace('game-', ''));
      if (chat_client_id) {
        const sender = game.getPlayers()?.get(chat_client_id) ?? game.getViewers()?.get(chat_client_id);
        game.addChat({
          message: message.content,
          sender: sender?.nickname ?? chat_client_id.toString(),
        });
        if (chat_client_id !== game.getConnectionMetadata()?.client_id) {
          Sounds.playTaunt(message.content);
        }
      }
      break;
    case 'game-player-disconnected':
      const disconnectee_id = parseInt(message.data);
      if (!!disconnectee_id) {
        game.playerDisconnected(disconnectee_id);
      }
      break;
    case 'game-player-connected':
      const connectee_id = parseInt(message.data);
      if (!!connectee_id) {
        game.playerConnected(connectee_id);
      }
      break;
    case 'game-start':
      game.startGame();
      break;
    case 'game-resend-waiting-room-ack':
      break; // still waiting on other players, nothing to do
    case 'game-update':
      handleGameUpdate(game, message);
      break;
    case 'game-setting': {
      // small private acknowledgement outside the sequenced update history
      const content = JSON.parse(message.content);
      trackSettingsRev(game, content.rev);
      void game.getGameEl()?.gameUpdate({ update_id: 0, kind: message.data, content });
      break;
    }
    case 'game-failed-update': // player did something incorrect
      if (
        [
          'set-auto-renew-failed',
          'set-gather-point-failed',
          'set-default-unit-behavior-failed',
          'bet-failed',
          'play-card-failed',
        ].includes(message.data)
      ) {
        void game.getGameEl()?.gameUpdate({
          update_id: 0,
          kind: message.data,
          content: JSON.parse(message.content),
        });
      }
      break;
    case 'game-last-update-id': {
      const game_base = game.getGame()?.game_base;
      const behind = parseInt(message.data) > (game_base?.highest_received_update_id ?? 0);
      const missed_setting = parseInt(message.content) > (game_base?.settings_rev ?? 0);
      if (behind) {
        game.socketSend(createMessage(`client-${game.getConnectionMetadata()?.client_id}`, 'game-resend-last-update'));
      } else if (missed_setting) {
        void game.resyncState();
      }
      break;
    }
    case 'game-connected-failed': // can be called at end of game before client realizes game is over
      break;
    case 'game-get-update-failed':
    case 'game-resend-last-update-failed':
    case 'game-resend-waiting-room-failed':
    case 'game-update-failed':
      errored = true;
      console.log(message.content);
      try {
        void game.recoverGame();
      } catch (e) {
        console.log(e);
      }
      break;
    default:
      errored = true;
      console.log('Unknown message type', message.kind, 'from', message.sender);
      break;
  }
  if (errored) {
    game.message_error_count++;
    if (game.message_error_count >= MAX_ERROR_COUNT) {
      game.dispatchEvent(
        new CustomEvent<string>('connection_lost', {
          detail: 'An error was encountered; please refresh your connection.',
        })
      );
    }
  } else {
    game.message_error_count = 0;
  }
}

async function handleGameUpdate(game: DwgGame, message: ServerMessage): Promise<void> {
  await game.refreshing;
  const game_ob = game.getGame();
  const game_el = game.getGameEl();
  if (!game_ob || !game_el) {
    return;
  }
  const sender_split = message.sender.split('-');
  if (sender_split.length < 3) {
    return;
  }
  const game_update_id = parseInt(sender_split[2]);
  if (!game_update_id) {
    return;
  }
  const game_base = game_ob.game_base;
  const current = (): boolean => game.getGame() === game_ob && game.getGameEl() === game_el && !game.refreshing;
  async function runUpdate(): Promise<void> {
    if (game.running_updates || !current()) {
      return;
    }
    game.running_updates = true;
    try {
      while (game.update_queue.length > 0 && current()) {
        const update = game.update_queue.shift()!;
        if (!game_base.persistant_history) {
          if (update.update_id <= (game_base.last_applied_update_id ?? 0)) {
            continue;
          }
          console.log(`Applying state-snapshot update id ${update.update_id}`);
          await game_el?.gameUpdate(update);
          if (!current()) {
            return;
          }
          game_base.last_applied_update_id = update.update_id;
          game_base.settings_rev = update.content?.game?.game_base?.settings_rev ?? game_base.settings_rev;
          continue;
        }
        let last_update_id = game_base?.last_applied_update_id ?? -2;
        // running update for next update
        if (update.update_id - 1 === last_update_id) {
          console.log(`Applying update id ${update.update_id}`);
          await game_el?.gameUpdate(update);
          if (!current()) {
            return;
          }
          game_base.last_applied_update_id = update.update_id;
          last_update_id = update.update_id;
          while (true) {
            const next_update = game_base.updates?.get(last_update_id + 1);
            if (!next_update) {
              break;
            }
            console.log(`Applying next update id ${next_update.update_id}`);
            await game_el?.gameUpdate(next_update);
            if (!current()) {
              return;
            }
            game_base.last_applied_update_id = next_update.update_id;
            last_update_id = next_update.update_id;
          }
        } else if (update.update_id - 1 > last_update_id) {
          game
            .getSocket()
            ?.send(
              createMessage(
                `client-${game.getConnectionMetadata()?.client_id}`,
                'game-get-update',
                '',
                `${last_update_id + 1}`
              )
            );
        } else {
        } // ignore updates that are already applied
        if ((game_base.highest_received_update_id ?? -2) > last_update_id) {
          // check if received higher update while updating
          game
            .getSocket()
            ?.send(
              createMessage(
                `client-${game.getConnectionMetadata()?.client_id}`,
                'game-get-update',
                '',
                `${last_update_id + 1}`
              )
            );
        }
      }
    } catch (e) {
      console.error('Error applying game update:', e);
      if (current()) {
        await game.recoverGame();
      }
    } finally {
      if (game.getGame() === game_ob && game.getGameEl() === game_el) {
        game.running_updates = false;
      }
    }
  }
  try {
    const parse_start = performance.now();
    const update_message: UpdateMessage = {
      update_id: game_update_id,
      kind: message.data,
      content: JSON.parse(message.content),
    };
    perfEvent('game_update', {
      update_id: game_update_id,
      update_kind: message.data,
      bytes: message.content.length,
      parse_ms: performance.now() - parse_start,
    });
    console.log(
      `Received game update ${game_update_id}${game_base.persistant_history ? `: ${JSON.stringify(update_message)}` : ''}`
    );
    if (game_base.persistant_history) {
      game_base.updates?.set(game_update_id, update_message);
    } else if (game_update_id > (game_base.highest_received_update_id ?? 0)) {
      // state snapshots replace each other, so only the newest is kept
      game_base.updates?.clear();
      game_base.updates?.set(game_update_id, update_message);
    }
    game_base.highest_received_update_id = Math.max(game_base.highest_received_update_id ?? 0, game_update_id);
    if (!game_base.persistant_history && game_update_id <= (game_base.last_applied_update_id ?? 0)) {
      return;
    }
    game.update_queue.push(update_message);
    if (game_base.game_started && !game_base.game_ended && game.getLaunched()) {
      runUpdate();
    }
  } catch (e) {
    console.log(e);
    if (game_base.persistant_history) {
      game
        .getSocket()
        ?.send(
          createMessage(`client-${game.getConnectionMetadata()?.client_id}`, 'game-get-update', '', `${game_update_id}`)
        );
    }
  }
}
