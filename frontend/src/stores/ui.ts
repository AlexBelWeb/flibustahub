import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const paletteOpen = ref(false)
  const onboardingOpen = ref(false)
  const onboardingDismissed = ref(false)
  const searchFocusNonce = ref(0)

  function openPalette() {
    paletteOpen.value = true
  }

  function closePalette() {
    paletteOpen.value = false
  }

  function togglePalette() {
    paletteOpen.value = !paletteOpen.value
  }

  function openOnboarding() {
    onboardingDismissed.value = false
    onboardingOpen.value = true
  }

  function skipOnboarding() {
    onboardingOpen.value = false
    onboardingDismissed.value = true
  }

  function requestSearchFocus() {
    searchFocusNonce.value += 1
  }

  return {
    paletteOpen,
    onboardingOpen,
    onboardingDismissed,
    searchFocusNonce,
    openPalette,
    closePalette,
    togglePalette,
    openOnboarding,
    skipOnboarding,
    requestSearchFocus,
  }
})
