import axios from 'axios'

export type ApiErrorCode =
  | 'validation_error'
  | 'login_already_exists'
  | 'invalid_credentials'
  | 'invalid_token'
  | 'user_not_found'
  | 'item_not_found'
  | 'category_not_found'
  | 'wanted_category_not_found'
  | 'wish_not_found'
  | 'not_found'
  | 'too_many_photos'
  | 'unsupported_image_type'
  | 'image_too_large'
  | 'file_storage_error'
  | 'internal_error'
  | 'unknown_error'

export class ApiError extends Error {
  code: ApiErrorCode

  constructor(code: ApiErrorCode, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

export function apiErrorFromAxios(error: unknown): ApiError {
  if (axios.isAxiosError(error) && error.response?.data?.error) {
    const { code, message } = error.response.data.error as { code: ApiErrorCode; message: string }
    return new ApiError(code, message)
  }
  return new ApiError('unknown_error', 'unexpected error')
}
