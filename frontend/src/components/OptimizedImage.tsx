/**
 * =============================================================================
 * Optimized Image Component
 * =============================================================================
 * Progressive image loading with lazy loading and format optimization
 */

import {
  useState,
  useEffect,
  useRef,
  useCallback,
  CSSProperties,
  ImgHTMLAttributes,
} from 'react';
import { cn } from '@utils/cn';
import {
  imageConfig,
  generateSrcSet,
  generateSizes,
  getConnectionConfig,
  prefersReducedMotion,
} from '@config/loading';

// =============================================================================
// Types
// =============================================================================

export interface OptimizedImageProps extends Omit<
  ImgHTMLAttributes<HTMLImageElement>,
  'src' | 'srcSet'
> {
  src: string;
  alt: string;
  width?: number;
  height?: number;
  aspectRatio?: string;
  priority?: boolean;
  placeholder?: 'blur' | 'color' | 'none';
  blurDataURL?: string;
  placeholderColor?: string;
  objectFit?: CSSProperties['objectFit'];
  objectPosition?: CSSProperties['objectPosition'];
  sizes?: string;
  quality?: number;
  onLoadingComplete?: (result: {
    naturalWidth: number;
    naturalHeight: number;
  }) => void;
  fallback?: string;
}

interface ImageState {
  isLoading: boolean;
  isLoaded: boolean;
  hasError: boolean;
  currentSrc: string | null;
}

// =============================================================================
// Hooks
// =============================================================================

/**
 * Intersection Observer hook for lazy loading
 */
function useIntersectionObserver(
  ref: React.RefObject<Element>,
  options: IntersectionObserverInit = {}
): boolean {
  const [isIntersecting, setIsIntersecting] = useState(false);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsIntersecting(true);
          observer.disconnect();
        }
      },
      {
        rootMargin: imageConfig.lazyOffset,
        ...options,
      }
    );

    observer.observe(element);

    return () => observer.disconnect();
  }, [ref, options]);

  return isIntersecting;
}

/**
 * Native lazy loading support check
 */
function supportsLazyLoading(): boolean {
  return 'loading' in HTMLImageElement.prototype;
}

// =============================================================================
// Component
// =============================================================================

export function OptimizedImage({
  src,
  alt,
  width,
  height,
  aspectRatio,
  priority = false,
  placeholder = 'blur',
  blurDataURL,
  placeholderColor = '#e5e7eb',
  objectFit = 'cover',
  objectPosition = 'center',
  sizes,
  quality = imageConfig.quality,
  onLoadingComplete,
  fallback,
  className,
  style,
  ...props
}: OptimizedImageProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const isInView = useIntersectionObserver(containerRef);
  const connectionConfig = getConnectionConfig();
  const reducedMotion = prefersReducedMotion();

  const [state, setState] = useState<ImageState>({
    isLoading: true,
    isLoaded: false,
    hasError: false,
    currentSrc: null,
  });

  // Determine if we should load
  const shouldLoad = priority || isInView;

  // Generate responsive sources
  const webpSrcSet = generateSrcSet(src, imageConfig.breakpoints, 'webp');
  const avifSrcSet = generateSrcSet(src, imageConfig.breakpoints, 'avif');
  const defaultSizes = sizes || generateSizes();

  // Determine placeholder style
  const getPlaceholderStyle = (): CSSProperties => {
    if (placeholder === 'none') return {};

    if (placeholder === 'color') {
      return { backgroundColor: placeholderColor };
    }

    if (placeholder === 'blur' && blurDataURL) {
      return {
        backgroundImage: `url(${blurDataURL})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        filter: 'blur(20px)',
        transform: 'scale(1.1)',
      };
    }

    return { backgroundColor: placeholderColor };
  };

  // Handle image load
  const handleLoad = useCallback(() => {
    setState((prev) => ({
      ...prev,
      isLoading: false,
      isLoaded: true,
    }));

    if (onLoadingComplete && imgRef.current) {
      onLoadingComplete({
        naturalWidth: imgRef.current.naturalWidth,
        naturalHeight: imgRef.current.naturalHeight,
      });
    }
  }, [onLoadingComplete]);

  // Handle image error
  const handleError = useCallback(() => {
    setState((prev) => ({
      ...prev,
      isLoading: false,
      hasError: true,
    }));

    // Try fallback if available
    if (fallback && imgRef.current && imgRef.current.src !== fallback) {
      imgRef.current.src = fallback;
    }
  }, [fallback]);

  // Preload high-priority images
  useEffect(() => {
    if (priority && typeof window !== 'undefined') {
      const link = document.createElement('link');
      link.rel = 'preload';
      link.as = 'image';
      link.href = src;
      link.imageSrcset = webpSrcSet;
      link.imageSizes = defaultSizes;
      document.head.appendChild(link);

      return () => {
        document.head.removeChild(link);
      };
    }
  }, [priority, src, webpSrcSet, defaultSizes]);

  // Container styles
  const containerStyle: CSSProperties = {
    position: 'relative',
    overflow: 'hidden',
    width: width ? `${width}px` : '100%',
    height: height ? `${height}px` : undefined,
    aspectRatio:
      aspectRatio || (width && height ? `${width}/${height}` : undefined),
    ...style,
  };

  // Image styles
  const imageStyle: CSSProperties = {
    objectFit,
    objectPosition,
    width: '100%',
    height: '100%',
    opacity: state.isLoaded ? 1 : 0,
    transition: reducedMotion
      ? 'none'
      : `opacity ${imageConfig.fadeInDuration}ms ease-in-out`,
  };

  // Placeholder styles
  const placeholderStyle: CSSProperties = {
    position: 'absolute',
    inset: 0,
    ...getPlaceholderStyle(),
    opacity: state.isLoaded ? 0 : 1,
    transition: reducedMotion
      ? 'none'
      : `opacity ${imageConfig.fadeInDuration}ms ease-in-out`,
  };

  // Error state
  if (state.hasError && !fallback) {
    return (
      <div
        ref={containerRef}
        className={cn(
          'bg-gray-100 flex items-center justify-center',
          className
        )}
        style={containerStyle}
      >
        <svg
          className="w-12 h-12 text-gray-300"
          fill="currentColor"
          viewBox="0 0 24 24"
        >
          <path d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={cn('relative', className)}
      style={containerStyle}
    >
      {/* Placeholder */}
      {placeholder !== 'none' && !state.isLoaded && (
        <div style={placeholderStyle} aria-hidden="true" />
      )}

      {/* Picture element for format selection */}
      {shouldLoad && (
        <picture>
          {/* AVIF source (best compression) */}
          {connectionConfig.loadHighResImages && (
            <source
              type="image/avif"
              srcSet={avifSrcSet}
              sizes={defaultSizes}
            />
          )}

          {/* WebP source (good compression, wide support) */}
          <source type="image/webp" srcSet={webpSrcSet} sizes={defaultSizes} />

          {/* Fallback img */}
          <img
            ref={imgRef}
            src={src}
            alt={alt}
            width={width}
            height={height}
            style={imageStyle}
            loading={priority ? 'eager' : 'lazy'}
            decoding={priority ? 'sync' : 'async'}
            fetchPriority={priority ? 'high' : 'auto'}
            onLoad={handleLoad}
            onError={handleError}
            {...props}
          />
        </picture>
      )}
    </div>
  );
}

// =============================================================================
// Background Image Component
// =============================================================================

interface BackgroundImageProps {
  src: string;
  blurDataURL?: string;
  className?: string;
  style?: CSSProperties;
  children?: React.ReactNode;
  overlay?: string;
  priority?: boolean;
}

export function BackgroundImage({
  src,
  blurDataURL,
  className,
  style,
  children,
  overlay,
  priority = false,
}: BackgroundImageProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const isInView = useIntersectionObserver(containerRef);
  const [isLoaded, setIsLoaded] = useState(false);
  const reducedMotion = prefersReducedMotion();

  const shouldLoad = priority || isInView;

  useEffect(() => {
    if (!shouldLoad) return;

    const img = new Image();
    img.onload = () => setIsLoaded(true);
    img.src = src;
  }, [shouldLoad, src]);

  const containerStyle: CSSProperties = {
    position: 'relative',
    ...style,
  };

  const imageStyle: CSSProperties = {
    position: 'absolute',
    inset: 0,
    backgroundImage: isLoaded
      ? `url(${src})`
      : blurDataURL
        ? `url(${blurDataURL})`
        : undefined,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    filter: isLoaded ? 'none' : 'blur(20px)',
    transform: isLoaded ? 'none' : 'scale(1.1)',
    transition: reducedMotion ? 'none' : 'filter 0.3s, transform 0.3s',
  };

  const overlayStyle: CSSProperties = overlay
    ? {
        position: 'absolute',
        inset: 0,
        backgroundColor: overlay,
      }
    : {};

  return (
    <div ref={containerRef} className={className} style={containerStyle}>
      <div style={imageStyle} aria-hidden="true" />
      {overlay && <div style={overlayStyle} aria-hidden="true" />}
      <div style={{ position: 'relative', zIndex: 1 }}>{children}</div>
    </div>
  );
}

// =============================================================================
// Image Gallery with Progressive Loading
// =============================================================================

interface GalleryImageProps {
  images: Array<{
    src: string;
    alt: string;
    blurDataURL?: string;
  }>;
  columns?: 1 | 2 | 3 | 4;
  gap?: number;
  aspectRatio?: string;
  className?: string;
}

export function ImageGallery({
  images,
  columns = 3,
  gap = 16,
  aspectRatio = '1/1',
  className,
}: GalleryImageProps) {
  const columnClasses = {
    1: 'grid-cols-1',
    2: 'grid-cols-2',
    3: 'grid-cols-3',
    4: 'grid-cols-4',
  };

  return (
    <div
      className={cn('grid', columnClasses[columns], className)}
      style={{ gap: `${gap}px` }}
    >
      {images.map((image, index) => (
        <OptimizedImage
          key={index}
          src={image.src}
          alt={image.alt}
          aspectRatio={aspectRatio}
          blurDataURL={image.blurDataURL}
          className="rounded-lg"
        />
      ))}
    </div>
  );
}

// =============================================================================
// Avatar with Optimized Loading
// =============================================================================

interface AvatarProps {
  src: string;
  alt: string;
  size?: 'sm' | 'md' | 'lg' | 'xl' | number;
  fallback?: string;
  className?: string;
}

const avatarSizes = {
  sm: 32,
  md: 40,
  lg: 48,
  xl: 64,
};

export function Avatar({
  src,
  alt,
  size = 'md',
  fallback,
  className,
}: AvatarProps) {
  const sizeValue = typeof size === 'number' ? size : avatarSizes[size];
  const [hasError, setHasError] = useState(false);

  // Generate initials from alt text
  const initials = alt
    .split(' ')
    .map((word) => word[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  if (hasError || !src) {
    return (
      <div
        className={cn(
          'rounded-full bg-gray-200 flex items-center justify-center text-gray-600 font-medium',
          className
        )}
        style={{
          width: sizeValue,
          height: sizeValue,
          fontSize: sizeValue * 0.4,
        }}
      >
        {initials}
      </div>
    );
  }

  return (
    <OptimizedImage
      src={src}
      alt={alt}
      width={sizeValue}
      height={sizeValue}
      className={cn('rounded-full', className)}
      placeholder="color"
      placeholderColor="#e5e7eb"
      fallback={fallback}
      onLoadingComplete={() => {}}
      onError={() => setHasError(true)}
    />
  );
}

// =============================================================================
// Preload Images
// =============================================================================

/**
 * Preload an array of images
 */
export function preloadImages(urls: string[]): Promise<void[]> {
  return Promise.all(
    urls.map(
      (url) =>
        new Promise<void>((resolve, reject) => {
          const img = new Image();
          img.onload = () => resolve();
          img.onerror = reject;
          img.src = url;
        })
    )
  );
}

/**
 * Create blur placeholder from image
 */
export async function createBlurPlaceholder(
  src: string,
  size = 10
): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';

    img.onload = () => {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');

      if (!ctx) {
        reject(new Error('Could not get canvas context'));
        return;
      }

      // Scale down
      const aspectRatio = img.width / img.height;
      canvas.width = size;
      canvas.height = Math.round(size / aspectRatio);

      // Draw scaled image
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);

      // Get data URL
      resolve(canvas.toDataURL('image/jpeg', 0.5));
    };

    img.onerror = reject;
    img.src = src;
  });
}
