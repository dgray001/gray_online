/** Class giving functionality of a map but with multiple keys */
export class MultiMap<K, V> {
  private keynames: string[] = [];
  private maps = new Map<string, Map<K, V>>();
  private _size = 0;

  constructor(keynames: string[]) {
    this.keynames = keynames;
    for (const keyname of keynames) {
      this.maps.set(keyname, new Map<K, V>());
    }
  }

  public get(keyname: string, key: K): V|undefined {
    return this.maps.get(keyname)?.get(key);
  }

  /** Returns whether the value was successfully set on the multikey map */
  public set(keys: K[], value: V): boolean {
    if (keys.length !== this.keynames.length || keys.length === 0) {
      return false;
    }
    for (const [i, keyname] of this.keynames.entries()) {
      this.maps.get(keyname).set(keys[i], value);
    }
    this._size = this.maps.get(this.keynames[0]).size;
    return true;
  }

  /** Returns whether the object was successfully deleted */
  public delete(keys: K[]): boolean {
    if (keys.length !== this.keynames.length || keys.length === 0) {
      return false;
    }
    let deleted = true;
    for (const [i, keyname] of this.keynames.entries()) {
      if (!this.maps.get(keyname).delete(keys[i])) {
        deleted = false;
      }
    }
    this._size = this.maps.get(this.keynames[0]).size;
    return deleted;
  }

  public clear() {
    for (const [i, keyname] of this.keynames.entries()) {
      this.maps.get(keyname).clear();
    }
    this._size = 0;
  }

  public size(): number {
    return this._size;
  }

  public keys(keyname: string): IterableIterator<K> {
    if (this.maps.has(keyname)) {
      return [].values();
    }
    return this.maps.get(keyname)?.keys();
  }

  public values(): IterableIterator<V> {
    if (this._size === 0) {
      return [].values();
    }
    return this.maps.get(this.keynames[0]).values();
  }
}
