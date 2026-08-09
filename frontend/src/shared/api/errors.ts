export type ApiErrorCode =
  | 'validation_error'
  | 'login_already_exists'
  | 'invalid_credentials'
  | 'invalid_token'
  | 'user_not_found'
  | 'item_not_found'
  | 'too_many_photos'
  | 'internal_error'

export class ApiError extends Error {
  code: ApiErrorCode

  constructor(code: ApiErrorCode, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}
