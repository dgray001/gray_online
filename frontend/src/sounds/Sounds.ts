export class Sounds {
  private static instance: Sounds = new Sounds();

  private sounds = new Map<string, Promise<HTMLAudioElement>>();

  private constructor() {}

  public static preload(s: string): Promise<HTMLAudioElement> {
    const cached = Sounds.instance.sounds.get(s);
    if (cached) {
      return cached;
    }
    const sound = new Audio();
    const loading = Sounds.waitUntilPlayable(sound, s).catch((error: unknown): never => {
      Sounds.instance.sounds.delete(s);
      throw error;
    });
    Sounds.instance.sounds.set(s, loading);
    return loading;
  }

  private static waitUntilPlayable(sound: HTMLAudioElement, name: string): Promise<HTMLAudioElement> {
    return new Promise((resolve, reject): void => {
      const listeners = new AbortController();
      sound.addEventListener(
        'canplay',
        (): void => {
          listeners.abort();
          resolve(sound);
        },
        { signal: listeners.signal }
      );
      const failed = (): void => {
        listeners.abort();
        reject(sound.error ?? new Error(`Unable to load sound ${name}`));
      };
      sound.addEventListener('error', failed, { signal: listeners.signal });
      sound.addEventListener('abort', failed, { signal: listeners.signal });
      sound.preload = 'auto';
      sound.src = `/sounds/${name}.mp3`;
    });
  }

  public static async play(s: string): Promise<void> {
    try {
      const sound = await Sounds.preload(s);
      await sound.play();
    } catch (e) {
      console.error(e);
    }
  }
}
