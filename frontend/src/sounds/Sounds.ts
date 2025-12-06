
export class Sounds {
  private static instance: Sounds = new Sounds();

  private sounds: Map<string, HTMLAudioElement> = new Map<string, HTMLAudioElement>();

  private constructor() {}

  public static async play(s: string) {
    try {
      let sound = Sounds.instance.sounds.get(s);
      if (!sound) {
        sound = new Audio(`sounds/${s}.mp3`);
        Sounds.instance.sounds.set(s, sound);
      }
      await sound.play();
    } catch (e) {
      console.error(e);
    }
  }
}
