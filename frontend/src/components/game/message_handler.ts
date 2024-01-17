import {ServerMessage, createMessage} from "../lobby/data_models";
import {UpdateMessage} from "./data_models";
import {DwgGame} from "./game";

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
  if (message.kind !== 'ping-update') {
    console.log(message);
  }
  switch(message.kind) {
    case "ping-update":
      // TODO: implement
      break;
    case "game-chat":
      console.log(message.sender, game.game?.game_base?.viewers);
      const chat_client_id = parseInt((message.sender ?? '').replace('game-', ''));
      if (chat_client_id) {
        const sender = game.game?.game_base?.players.get(chat_client_id) ?? game.game?.game_base?.viewers.get(chat_client_id);
        game.chatbox.addChat({
          message: message.content,
          sender: sender?.nickname ?? chat_client_id.toString(),
        });
      }
      break;
    case "game-player-connected":
      const connectee_id = parseInt(message.data);
      if (connectee_id) {
        const el = game.players_waiting_els.get(connectee_id);
        if (!!el) {
          el.innerText = 'Connected';
          el.classList.add('connected');
        }
        const player = game.game?.game_base?.players.get(connectee_id);
        if (player) {
          player.connected = true;
        }
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
      console.log(message.content);
      try {
        game.refreshGame();
      } catch(e) {
        console.log(e);
      }
      break;
    default:
      console.log("Unknown message type", message.kind, "from", message.sender);
      break;
  }
}

let running_updates = false;
async function handleGameUpdate(game: DwgGame, message: ServerMessage) {
  if (!game.game_el) {
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
    if (update.update_id - 1 === game.game.game_base.last_continuous_update_id) {
      game.game.game_base.last_continuous_update_id = update.update_id;
      console.log(`applying update id ${update.update_id}`);
      await game.game_el.gameUpdate(update);
      while (game.game.game_base.updates.has(game.game.game_base.last_continuous_update_id + 1)) {
        const nextUpdate = game.game.game_base.updates.get(game.game.game_base.last_continuous_update_id + 1);
        game.game.game_base.last_continuous_update_id = nextUpdate.update_id;
        console.log(`applying update id ${nextUpdate.update_id}`);
        await game.game_el.gameUpdate(nextUpdate);
      }
    } else if (update.update_id - 1 > game.game.game_base.last_continuous_update_id) {
      game.socket.send(createMessage(
        `client-${game.connection_metadata.client_id}`,
        'game-get-update',
        '',
        `${game.game.game_base.last_continuous_update_id + 1}`,
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
    game.game.game_base.updates.set(game_update_id, updateMessage);
    if (!running_updates) {
      runUpdate(updateMessage);
    }
  } catch(e) {
    console.log(e);
    game.socket.send(createMessage(
      `client-${game.connection_metadata.client_id}`,
      'game-get-update',
      '',
      `${game_update_id}`,
    ));
  }
}
