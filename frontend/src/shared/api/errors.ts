export type ApiErrorCode =
  | 'validation_error'
  | 'login_already_exists'
  | 'invalid_credentials'
  | 'invalid_token'
  | 'user_not_found'
  | 'internal_error'

export class ApiError extends Error {
  code: ApiErrorCode

  constructor(code: ApiErrorCode, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}
