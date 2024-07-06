export declare class MultiMap<K, V> {
    private keynames;
    private maps;
    private _size;
    constructor(keynames: string[]);
    get(keyname: string, key: K): V | undefined;
    set(keys: K[], value: V): boolean;
    delete(keys: K[]): boolean;
    clear(): void;
    size(): number;
    keys(keyname: string): IterableIterator<K>;
    values(): IterableIterator<V>;
}
