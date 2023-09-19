import {ServerMessage} from "../lobby/data_models";
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
    return;
  }
  if (message.kind !== 'ping-update') {
    console.log(message);
  }
  switch(message.kind) {
    case "ping-update":
      // TODO: implement
      break;
    default:
      console.log("Unknown message type", message.kind, "from", message.sender);
      break;
  }
}