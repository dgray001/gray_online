export interface ApiResponse<T> {
    success: boolean;
    result?: T;
    error_message?: string;
}
export declare function websocketPath(): string;
export declare function apiGet<T>(api: string): Promise<ApiResponse<T>>;
export declare function apiPost<T>(api: string, data: any): Promise<ApiResponse<T>>;
