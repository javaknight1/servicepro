import api from './api';

export interface ProfilePictureUploadResponse {
  profile_picture_url: string;
}

export const userService = {
  /**
   * Upload a profile picture
   * @param blob - The cropped image blob
   * @returns The URL of the uploaded profile picture
   */
  uploadProfilePicture: async (blob: Blob): Promise<string> => {
    const formData = new FormData();
    formData.append('file', blob, 'profile-picture.jpg');

    const response = await api.post<ProfilePictureUploadResponse>(
      '/v1/users/me/profile-picture',
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );

    return response.data.profile_picture_url;
  },

  /**
   * Delete the current user's profile picture
   */
  deleteProfilePicture: async (): Promise<void> => {
    await api.delete('/v1/users/me/profile-picture');
  },
};
