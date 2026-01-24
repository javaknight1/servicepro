import { useState, useRef } from 'react';
import { Avatar, Button, ImageCropModal } from '@components/shared';
import { useAuthStore } from '@store';
import { userService } from '@services/userService';
import { Upload, Trash2 } from 'lucide-react';

const MAX_FILE_SIZE = 2 * 1024 * 1024; // 2MB
const ALLOWED_TYPES = ['image/jpeg', 'image/png'];

export function ProfilePictureSection() {
  const user = useAuthStore((state) => state.user);
  const updateProfilePicture = useAuthStore(
    (state) => state.updateProfilePicture
  );

  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [showCropModal, setShowCropModal] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reset error
    setError(null);

    // Validate file type
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError('Please select a JPEG or PNG image');
      return;
    }

    // Validate file size
    if (file.size > MAX_FILE_SIZE) {
      setError('File size must be less than 2MB');
      return;
    }

    // Read file and show crop modal
    const reader = new FileReader();
    reader.onload = () => {
      setImageSrc(reader.result as string);
      setShowCropModal(true);
    };
    reader.readAsDataURL(file);

    // Reset input so the same file can be selected again
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleCropComplete = async (croppedBlob: Blob) => {
    setIsUploading(true);
    setError(null);

    try {
      const url = await userService.uploadProfilePicture(croppedBlob);
      updateProfilePicture(url);
      setShowCropModal(false);
      setImageSrc(null);
    } catch (err) {
      console.error('Failed to upload profile picture:', err);
      setError('Failed to upload profile picture. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };

  const handleDelete = async () => {
    if (!user?.profile_picture_url) return;

    if (!confirm('Are you sure you want to remove your profile picture?')) {
      return;
    }

    setIsDeleting(true);
    setError(null);

    try {
      await userService.deleteProfilePicture();
      updateProfilePicture(null);
    } catch (err) {
      console.error('Failed to delete profile picture:', err);
      setError('Failed to delete profile picture. Please try again.');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleCloseCropModal = () => {
    setShowCropModal(false);
    setImageSrc(null);
  };

  if (!user) return null;

  return (
    <div className="mb-6 pb-6 border-b border-neutral-200">
      <h3 className="text-sm font-medium text-neutral-900 mb-4">
        Profile Picture
      </h3>

      <div className="flex items-center gap-6">
        <Avatar
          email={user.email}
          profilePictureUrl={user.profile_picture_url}
          size="xl"
        />

        <div className="flex flex-col gap-2">
          <div className="flex gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png"
              onChange={handleFileSelect}
              className="hidden"
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={isUploading}
            >
              <Upload className="h-4 w-4 mr-2" />
              {user.profile_picture_url ? 'Change' : 'Upload'}
            </Button>

            {user.profile_picture_url && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleDelete}
                disabled={isDeleting}
                className="text-red-600 hover:text-red-700 hover:bg-red-50"
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Remove
              </Button>
            )}
          </div>

          <p className="text-xs text-neutral-500">JPEG or PNG, max 2MB</p>

          {error && <p className="text-xs text-red-600">{error}</p>}
        </div>
      </div>

      {imageSrc && (
        <ImageCropModal
          isOpen={showCropModal}
          onClose={handleCloseCropModal}
          imageSrc={imageSrc}
          onCropComplete={handleCropComplete}
        />
      )}
    </div>
  );
}
