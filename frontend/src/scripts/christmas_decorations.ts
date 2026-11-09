import { Sounds } from '../sounds/Sounds';

const BELL_SVG =
  '<svg viewBox="0 0 32 32" width="26" height="26">' +
  '<path d="M12 4 C10 1, 6 3, 7 8 C4 7, 2 11, 6 13 C10 14, 13 9, 12 4 Z" fill="#2d6e35"/>' +
  '<path d="M20 4 C22 1, 26 3, 25 8 C28 7, 30 11, 26 13 C22 14, 19 9, 20 4 Z" fill="#2d6e35"/>' +
  '<circle cx="14" cy="7" r="2.5" fill="#c42020"/>' +
  '<circle cx="18" cy="7" r="2.5" fill="#c42020"/>' +
  '<circle cx="16" cy="10" r="2.5" fill="#e03030"/>' +
  '<path d="M16 10 C12 11, 9 14, 9 20 L7 23 L25 23 L23 20 C23 14, 20 11, 16 10 Z" fill="#f5c030" stroke="#b8860b" stroke-width="1"/>' +
  '<ellipse cx="16" cy="23" rx="9" ry="2.5" fill="#e5af20"/>' +
  '<circle cx="16" cy="24.5" r="2" fill="#7a4505"/>' +
  '</svg>';

function ringBell(button: HTMLElement): void {
  button.classList.remove('ringing');
  void button.offsetWidth;
  button.classList.add('ringing');
  void Sounds.play('jingle_bell');
}

export function createBell(size: number, id?: string): HTMLButtonElement {
  const button = document.createElement('button');
  if (id) {
    button.id = id;
  }
  button.className = 'christmas-bell';
  button.innerHTML = BELL_SVG.replace('width="26" height="26"', `width="${size}" height="${size}"`);
  button.title = 'Ring Bell';
  button.addEventListener('click', (): void => ringBell(button));
  return button;
}

function attachLobbyBells(): void {
  const create_btn = document.getElementById('create-room-button');
  if (create_btn && !document.getElementById('large-lobby-bell')) {
    const large_bell = createBell(42, 'large-lobby-bell');
    create_btn.insertAdjacentElement('afterend', large_bell);
  }
  const page_hdr = document.getElementById('page-header');
  if (page_hdr && !document.getElementById('header-small-bell')) {
    const small_bell = createBell(22, 'header-small-bell');
    page_hdr.appendChild(small_bell);
  }
}

const SLEIGH_SVG =
  '<svg viewBox="0 0 140 40" width="140" height="40">' +
  '<path d="M10 22 Q18 16 26 22 L30 18 Q32 14 36 17 L34 23 Q28 27 20 27 Z" fill="#964b00"/>' +
  '<circle cx="36" cy="17" r="1.5" fill="#e52b2b"/>' +
  '<path d="M45 22 Q53 16 61 22 L65 18 Q67 14 71 17 L69 23 Q63 27 55 27 Z" fill="#964b00"/>' +
  '<circle cx="71" cy="17" r="1.5" fill="#e52b2b"/>' +
  '<path d="M72 23 Q90 24 100 24" stroke="#d4af37" stroke-width="1" fill="none"/>' +
  '<path d="M98 27 Q100 15 116 16 Q122 14 128 20 Q135 25 130 30 Q118 34 102 31 Z" fill="#c41e3a"/>' +
  '<path d="M96 33 Q114 35 134 31" stroke="#d4af37" stroke-width="2" fill="none"/>' +
  '<circle cx="118" cy="15" r="4" fill="#ffffff"/>' +
  '<circle cx="119" cy="16" r="3" fill="#e52b2b"/>' +
  '</svg>';

function initSantaSleigh(): void {
  if (document.getElementById('santa-sleigh')) {
    return;
  }
  const sleigh = document.createElement('div');
  sleigh.id = 'santa-sleigh';
  sleigh.innerHTML = SLEIGH_SVG;
  sleigh.title = "Santa's Sleigh! Click to jingle";
  sleigh.addEventListener('click', (): void => {
    void Sounds.play('jingle_bell');
  });
  document.body.appendChild(sleigh);

  const launch_flight = (): void => {
    sleigh.classList.remove('flying');
    void sleigh.offsetWidth;
    sleigh.classList.add('flying');
  };
  launch_flight();
  setInterval(launch_flight, 32000);
}

export function initChristmasDecorations(): void {
  if (!document.getElementById('christmas-bell-widget')) {
    const corner_bell = createBell(28, 'christmas-bell-widget');
    document.body.appendChild(corner_bell);
  }
  attachLobbyBells();
  initSantaSleigh();
  setInterval(attachLobbyBells, 1500);
}
