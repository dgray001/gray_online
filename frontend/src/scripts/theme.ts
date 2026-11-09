import { startSnow } from './snow';
import { initChristmasDecorations } from './christmas_decorations';

interface ThemeDomainRule {
  suffix: string;
  theme: string;
}

const DOMAIN_THEMES: ThemeDomainRule[] = [{ suffix: '.christmas', theme: 'christmas' }];

export function detectTheme(hostname: string = window.location.hostname): string | undefined {
  const match = DOMAIN_THEMES.find((rule: ThemeDomainRule): boolean => hostname.endsWith(rule.suffix));
  return match?.theme;
}

export function initTheme(): void {
  const theme = detectTheme();
  if (theme) {
    document.body.dataset.theme = theme;
    if (theme === 'christmas') {
      startSnow();
      initChristmasDecorations();
    }
  }
}
