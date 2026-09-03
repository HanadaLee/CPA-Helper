import {
  currentLanguage,
  isEnglish,
  localize,
  setLanguage,
  toggleLanguage,
  useLanguagePreference,
} from './language'
import { copiedText, errorText, localizedCredentialServerMessage, localizedKeeperStatusDetail, localizedServerMessage } from './messages'

export {
  currentLanguage,
  isEnglish,
  localize,
  setLanguage,
  toggleLanguage,
  useLanguagePreference,
  type AppLanguage,
} from './language'
export { copiedText, errorText, localizedApiErrorMessage, localizedCredentialServerMessage, localizedKeeperStatusDetail, localizedServerMessage } from './messages'

export function useI18n() {
  return {
    copiedText,
    credentialServerText: localizedCredentialServerMessage,
    currentLanguage,
    errorText,
    isEnglish,
    keeperStatusText: localizedKeeperStatusDetail,
    language: currentLanguage,
    localize,
    serverText: localizedServerMessage,
    setLanguage,
    t: localize,
    toggleLanguage,
    useLanguagePreference,
  }
}
