import api from './api';
import type { User } from '@app-types';

export interface ProfilePictureUploadResponse {
  profile_picture_url: string;
}

export interface UpdateProfileRequest {
  first_name?: string | null;
  last_name?: string | null;
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

  /**
   * Update the current user's profile (first name and last name)
   * @param data - The profile data to update
   * @returns The updated user
   */
  updateProfile: async (data: UpdateProfileRequest): Promise<User> => {
    const response = await api.patch<User>('/v1/users/me/profile', data);
    return response.data;
  },
};
