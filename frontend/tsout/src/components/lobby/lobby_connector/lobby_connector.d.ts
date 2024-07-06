import { DwgElement } from '../../dwg_element';
import { ConnectionMetadata } from '../data_models';
import './lobby_connector.scss';
export declare interface ConnectData {
    nickname: string;
    try_reconnect: boolean;
    client_id?: number;
}
export declare const PREVIOUS_NICKNAME = "!!previous!!";
export declare class DwgLobbyConnector extends DwgElement {
    card: HTMLDivElement;
    reconnect_wrapper: HTMLDivElement;
    previous_nickname: HTMLDivElement;
    reconnect_button: HTMLButtonElement;
    new_connection_button: HTMLButtonElement;
    connect_wrapper: HTMLDivElement;
    nickname: HTMLInputElement;
    connect_button: HTMLButtonElement;
    status_message: HTMLDivElement;
    reconnect_data: ConnectData;
    constructor();
    protected parsedCallback(): Promise<void>;
    tryReconnecting(message: string, connection_metadata: ConnectionMetadata): void;
    connect(connect_data: ConnectData): void;
    invalid_names: string[];
    private validateInputName;
    private validateName;
}
