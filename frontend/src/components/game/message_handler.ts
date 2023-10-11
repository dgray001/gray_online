import {ServerMessage, createMessage} from "../lobby/data_models";
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
      const chat_client_id = parseInt(message.sender.replace('game-', ''));
      if (chat_client_id) {
        const sender = game.game?.game_base?.players.get(chat_client_id) ?? game.game?.game_base?.viewers.get(chat_client_id);
        game.chatbox.addChat({
          message: message.content,
          sender: sender.nickname ?? chat_client_id.toString(),
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
      game.game.game_base.game_started = true;
      game.waiting_room.classList.add('hide');
      game.game_container.classList.add('show');
      break;
    case "game-update":
      handleGameUpdate(game, message);
      break;
    case "game-connected-failed":
    case "game-update-failed":
    case "game-get-update-failed":
      // TODO: sync state
      break;
    default:
      console.log("Unknown message type", message.kind, "from", message.sender);
      break;
  }
}

function handleGameUpdate(game: DwgGame, message: ServerMessage) {
  const sender_split = message.sender.split('-');
  if (sender_split.length < 3) {
    return;
  }
  const game_update_id = parseInt(sender_split[2]);
  if (!game_update_id) {
    return;
  }
  try {
    const update = JSON.parse(message.content);
    const updateMessage = {
      update_id: game_update_id,
      kind: message.data,
      update,
    };
    game.game.game_base.updates.set(game_update_id, updateMessage);
    if (game_update_id - 1 === game.game.game_base.last_continuous_update_id) {
      game.game.game_base.last_continuous_update_id = game_update_id;
      game.game_el.gameUpdate(updateMessage);
      while (game.game.game_base.updates.has(game.game.game_base.last_continuous_update_id + 1)) {
        const nextUpdateMessage = game.game.game_base.updates.get(game.game.game_base.last_continuous_update_id + 1);
        game.game.game_base.last_continuous_update_id = nextUpdateMessage.update_id;
        game.game_el.gameUpdate(nextUpdateMessage);
      }
    } else if (game_update_id - 1 > game.game.game_base.last_continuous_update_id) {
      game.socket.send(createMessage(
        `client-${game.connection_metadata.client_id}`,
        'game-get-update',
        '',
        `${game.game.game_base.last_continuous_update_id + 1}`,
      ));
    } else {} // ignore updates that are already applied
  } catch(e) {
    game.socket.send(createMessage(
      `client-${game.connection_metadata.client_id}`,
      'game-get-update',
      '',
      `${game_update_id}`,
    ));
  }
}