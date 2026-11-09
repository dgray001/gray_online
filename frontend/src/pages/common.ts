// Script run by all pages

// Global style
import '../styles.scss';
import { clientOnMobile } from '../scripts/util';
import { initTheme } from '../scripts/theme';

initTheme();
document.body.classList.toggle('mobile', clientOnMobile());

// Trick iOS Safari to work with touch events and pseudo classes
document.body.ontouchstart = () => {};
