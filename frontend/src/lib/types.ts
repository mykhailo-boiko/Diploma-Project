export type Role =
  | "admin"
  | "warehouse_manager"
  | "logistics_manager"
  | "analyst"
  | "operator";

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: Role;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  data: {
    access_token: string;
    refresh_token: string;
    user: User;
  };
}
