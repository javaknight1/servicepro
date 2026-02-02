export interface User {
  id: string;
  email: string;
  role: 'user' | 'admin';
  first_name?: string;
  last_name?: string;
  profile_picture_url?: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

// LoginResponse is now cookie-based - no tokens in body
export interface LoginResponse {
  message: string;
  expiresIn: number; // in seconds
}

export interface RegisterRequest {
  email: string;
  password: string;
  invitation_token?: string;
}

export interface RegisterResponse {
  id: string;
  email: string;
  role: 'user' | 'admin';
  createdAt: string;
}

export interface PasswordResetRequestRequest {
  email: string;
}

export interface PasswordResetConfirmRequest {
  token: string;
  new_password: string;
}

export interface VerifyEmailRequest {
  token: string;
}

export interface ResendVerificationRequest {
  email: string;
}

export interface UpdateUserRequest {
  first_name?: string;
  last_name?: string;
  profile_picture_url?: string | null;
}

export interface ErrorResponse {
  error: string;
  message?: string;
}

export interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (
    email: string,
    password: string,
    invitationToken?: string
  ) => Promise<void>;
  logout: () => Promise<void>;
  refreshAccessToken: () => Promise<void>;
  setUser: (user: User) => void;
  updateProfilePicture: (url: string | null) => void;
  fetchCurrentUser: () => Promise<void>;
  checkAuth: () => Promise<boolean>;
}
