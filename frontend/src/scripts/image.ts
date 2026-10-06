const DEFAULT_IMAGE_URL = '/images/default.png';
const DEFAULT_IMAGE_DATA =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGNgYGAAAAAEAAH2FzhVAAAAAElFTkSuQmCC';

export function configureImageFallback(image: HTMLImageElement): void {
  image.draggable = false;
  image.addEventListener(
    'error',
    (): void => {
      image.addEventListener(
        'error',
        (): void => {
          image.src = DEFAULT_IMAGE_DATA;
        },
        { once: true }
      );
      image.src = DEFAULT_IMAGE_URL;
    },
    { once: true }
  );
}

export function createImage(url: string): HTMLImageElement {
  const image = document.createElement('img');
  configureImageFallback(image);
  image.src = url;
  return image;
}

export function isImageReady(image: HTMLImageElement): boolean {
  return image.complete && image.naturalWidth > 0 && image.naturalHeight > 0;
}

export function resolveImage(image: HTMLImageElement): HTMLImageElement {
  if (image.complete && !isImageReady(image)) {
    image.src = image.getAttribute('src') === DEFAULT_IMAGE_URL ? DEFAULT_IMAGE_DATA : DEFAULT_IMAGE_URL;
  }
  return image;
}
