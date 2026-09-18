import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { reactive, watchEffect } from 'vue'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Local stand-in for vueuse `reactiveOmit` so copied kit sources do not add a dependency. */
export function reactiveOmit<T extends Record<string, unknown>, K extends keyof T>(
  obj: T,
  ...keys: K[]
): Omit<T, K> {
  const excluded = new Set(keys as string[])
  const dest = reactive({} as Record<string, unknown>)
  watchEffect(() => {
    for (const key of Object.keys(dest)) {
      if (excluded.has(key) || !(key in obj)) {
        delete dest[key]
      }
    }
    for (const [key, value] of Object.entries(obj)) {
      if (!excluded.has(key)) {
        dest[key] = value
      }
    }
  })
  return dest as Omit<T, K>
}
