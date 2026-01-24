import { useState, useCallback } from 'react';
import Cropper from 'react-easy-crop';
import type { Area, Point } from 'react-easy-crop';
import { Modal } from '../Modal';
import { Button } from '../Button';
import { RotateCw } from 'lucide-react';

export interface ImageCropModalProps {
  isOpen: boolean;
  onClose: () => void;
  imageSrc: string;
  onCropComplete: (croppedBlob: Blob) => void;
}

async function getCroppedImg(
  imageSrc: string,
  pixelCrop: Area,
  rotation: number = 0
): Promise<Blob> {
  const image = await createImage(imageSrc);
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');

  if (!ctx) {
    throw new Error('Could not get canvas context');
  }

  const rotRad = (rotation * Math.PI) / 180;

  // Calculate bounding box of the rotated image
  const { width: bBoxWidth, height: bBoxHeight } = rotateSize(
    image.width,
    image.height,
    rotation
  );

  // Set canvas size to match the bounding box
  canvas.width = bBoxWidth;
  canvas.height = bBoxHeight;

  // Translate canvas context to a central location to allow rotating and flipping around the center
  ctx.translate(bBoxWidth / 2, bBoxHeight / 2);
  ctx.rotate(rotRad);
  ctx.translate(-image.width / 2, -image.height / 2);

  // Draw rotated image
  ctx.drawImage(image, 0, 0);

  // croppedAreaPixels values are based on the original image size
  // Extract the cropped image using these values
  const data = ctx.getImageData(
    pixelCrop.x,
    pixelCrop.y,
    pixelCrop.width,
    pixelCrop.height
  );

  // Set canvas width to final desired crop size
  canvas.width = pixelCrop.width;
  canvas.height = pixelCrop.height;

  // Place the data back
  ctx.putImageData(data, 0, 0);

  // Convert canvas to Blob
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob) {
          resolve(blob);
        } else {
          reject(new Error('Canvas is empty'));
        }
      },
      'image/jpeg',
      0.9
    );
  });
}

function createImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.addEventListener('load', () => resolve(image));
    image.addEventListener('error', (error) => reject(error));
    image.src = url;
  });
}

function rotateSize(width: number, height: number, rotation: number) {
  const rotRad = (rotation * Math.PI) / 180;
  return {
    width:
      Math.abs(Math.cos(rotRad) * width) + Math.abs(Math.sin(rotRad) * height),
    height:
      Math.abs(Math.sin(rotRad) * width) + Math.abs(Math.cos(rotRad) * height),
  };
}

export function ImageCropModal({
  isOpen,
  onClose,
  imageSrc,
  onCropComplete,
}: ImageCropModalProps) {
  const [crop, setCrop] = useState<Point>({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<Area | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);

  const onCropChange = useCallback((location: Point) => {
    setCrop(location);
  }, []);

  const onZoomChange = useCallback((newZoom: number) => {
    setZoom(newZoom);
  }, []);

  const onCropCompleteCallback = useCallback(
    (_croppedArea: Area, croppedAreaPixels: Area) => {
      setCroppedAreaPixels(croppedAreaPixels);
    },
    []
  );

  const handleRotate = useCallback(() => {
    setRotation((prev) => (prev + 90) % 360);
  }, []);

  const handleSave = useCallback(async () => {
    if (!croppedAreaPixels) return;

    setIsProcessing(true);
    try {
      const croppedBlob = await getCroppedImg(
        imageSrc,
        croppedAreaPixels,
        rotation
      );
      onCropComplete(croppedBlob);
      onClose();
    } catch (error) {
      console.error('Error cropping image:', error);
    } finally {
      setIsProcessing(false);
    }
  }, [croppedAreaPixels, imageSrc, rotation, onCropComplete, onClose]);

  const handleClose = useCallback(() => {
    // Reset state when closing
    setCrop({ x: 0, y: 0 });
    setZoom(1);
    setRotation(0);
    setCroppedAreaPixels(null);
    onClose();
  }, [onClose]);

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Crop Profile Picture">
      <div className="space-y-4">
        <div className="relative h-80 bg-neutral-900 rounded-lg overflow-hidden">
          <Cropper
            image={imageSrc}
            crop={crop}
            zoom={zoom}
            rotation={rotation}
            aspect={1}
            cropShape="round"
            showGrid={false}
            onCropChange={onCropChange}
            onZoomChange={onZoomChange}
            onCropComplete={onCropCompleteCallback}
          />
        </div>

        <div className="space-y-3">
          <div className="flex items-center gap-4">
            <label className="text-sm font-medium text-neutral-700 w-16">
              Zoom
            </label>
            <input
              type="range"
              min={1}
              max={3}
              step={0.1}
              value={zoom}
              onChange={(e) => setZoom(Number(e.target.value))}
              className="flex-1 h-2 bg-neutral-200 rounded-lg appearance-none cursor-pointer accent-primary-600"
            />
          </div>

          <div className="flex items-center gap-4">
            <label className="text-sm font-medium text-neutral-700 w-16">
              Rotate
            </label>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleRotate}
            >
              <RotateCw className="h-4 w-4 mr-2" />
              90°
            </Button>
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-4">
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            onClick={handleSave}
            disabled={isProcessing}
          >
            {isProcessing ? 'Processing...' : 'Save'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
