import {ServerMessage, createMessage} from "../lobby/data_models";
import {UpdateMessage} from "./data_models";
import {DwgGame} from "./game";

/** After this many repeated errors the frontend should shutdown */
const MAX_ERROR_COUNT = 3;
let error_count = 0;

function isGameMessage(kind: string): boolean {
  const lobby_prefixes = ['game', 'ping'];
  return !lobby_prefixes.every(prefix => !kind.startsWith(`${prefix}-`));
}

/** Handles messages for the frontend lobby */
export function handleMessage(game: DwgGame, message: ServerMessage) {
  if (!game.socketActive()) {
    return;
  }
  if (!isGameMessage(message.kind)) {
    console.log("not a game message", message);
    return;
  }
  let errored = false;
  if (message.kind !== 'ping-update') {
    console.log(message);
  }
  switch(message.kind) {
    case "ping-update":
      // TODO: implement
      return; // doesn't affect error count
    case "game-chat":
      const chat_client_id = parseInt((message.sender ?? '').replace('game-', ''));
      if (chat_client_id) {
        const sender = game.getPlayers()?.get(chat_client_id) ?? game.getViewers()?.get(chat_client_id);
        game.addChat({
          message: message.content,
          sender: sender?.nickname ?? chat_client_id.toString(),
        });
      }
      break;
    case "game-player-connected":
      const connectee_id = parseInt(message.data);
      if (!!connectee_id) {
        game.playerConnected(connectee_id);
      }
      break;
    case "game-start":
      game.startGame();
      break;
    case "game-update":
      handleGameUpdate(game, message);
      break;
    case "game-connected-failed":
    case "game-update-failed":
    case "game-get-update-failed":
    case "game-resend-last-update-failed":
    case "game-resend-waiting-room-failed":
      errored = true;
      console.log(message.content);
      try {
        game.refreshGame();
      } catch(e) {
        console.log(e);
      }
      break;
    default:
      errored = true;
      console.log("Unknown message type", message.kind, "from", message.sender);
      break;
  }
  if (errored) {
    error_count++;
    if (error_count >= MAX_ERROR_COUNT) {
      game.dispatchEvent(new CustomEvent('connection_lost', {
        detail: 'An error was encountered; please refresh your connection.'}));
    }
  } else {
    error_count = 0;
  }
}

let running_updates = false;
async function handleGameUpdate(game: DwgGame, message: ServerMessage) {
  if (!game.getGameEl() || !game.getGame) {
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
  async function runUpdate(update: UpdateMessage) {
    running_updates = true;
    if (update.update_id - 1 === game.getGame().game_base.last_continuous_update_id) {
      game.getGame().game_base.last_continuous_update_id = update.update_id;
      console.log(`applying update id ${update.update_id}`);
      await game.getGameEl().gameUpdate(update);
      while (game.getGame().game_base.updates.has(game.getGame().game_base.last_continuous_update_id + 1)) {
        const nextUpdate = game.getGame().game_base.updates.get(game.getGame().game_base.last_continuous_update_id + 1);
        game.getGame().game_base.last_continuous_update_id = nextUpdate.update_id;
        console.log(`applying update id ${nextUpdate.update_id}`);
        await game.getGameEl().gameUpdate(nextUpdate);
      }
    } else if (update.update_id - 1 > game.getGame().game_base.last_continuous_update_id) {
      game.getSocket().send(createMessage(
        `client-${game.getConnectionMetadata().client_id}`,
        'game-get-update',
        '',
        `${game.getGame().game_base.last_continuous_update_id + 1}`,
      ));
    } else {} // ignore updates that are already applied
    running_updates = false;
  }
  try {
    const update = JSON.parse(message.content);
    const updateMessage = {
      update_id: game_update_id,
      kind: message.data,
      update,
    };
    game.getGame().game_base.updates.set(game_update_id, updateMessage);
    if (!running_updates && game.getGame().game_base.game_started && !game.getGame().game_base.game_ended && game.getLaunched()) {
      runUpdate(updateMessage);
    }
  } catch(e) {
    console.log(e);
    game.getSocket().send(createMessage(
      `client-${game.getConnectionMetadata().client_id}`,
      'game-get-update',
      '',
      `${game_update_id}`,
    ));
  }
}
