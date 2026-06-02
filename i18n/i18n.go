// Package i18n provides internationalization for RyxoGo apps.
// Reactive locale switching — change language and all components re-render.
package i18n

import (
	"strings"
	"sync"

	"github.com/ahmad-nexarapp/ryxogo/signal"
)

// Translations maps a key to a translated string for one locale.
type Translations map[string]string

// I18n holds all translations and the current locale.
type I18n struct {
	mu       sync.RWMutex
	locales  map[string]Translations
	fallback string
	current  *signal.Signal[string] // reactive — switching re-renders UI
}

// global instance
var instance *I18n

// Setup initializes the i18n system with a default locale.
//
//	i18n.Setup("en")
//	i18n.Add("en", i18n.Translations{"greeting": "Hello"})
//	i18n.Add("es", i18n.Translations{"greeting": "Hola"})
func Setup(defaultLocale string) {
	instance = &I18n{
		locales:  make(map[string]Translations),
		fallback: defaultLocale,
		current:  signal.New(defaultLocale),
	}
}

// Add registers translations for a locale.
func Add(locale string, trans Translations) {
	if instance == nil {
		Setup("en")
	}
	instance.mu.Lock()
	if instance.locales[locale] == nil {
		instance.locales[locale] = make(Translations)
	}
	for k, v := range trans {
		instance.locales[locale][k] = v
	}
	instance.mu.Unlock()
}

// SetLocale changes the active locale — triggers re-render of all components
// that called T() during their render.
func SetLocale(locale string) {
	if instance == nil {
		return
	}
	instance.current.Set(locale)
}

// Locale returns the current locale (reactive — subscribes the caller).
func Locale() string {
	if instance == nil {
		return "en"
	}
	return instance.current.Val()
}

// T translates a key for the current locale.
// Reactive — when locale changes via SetLocale, components using T re-render.
//
//	rx.Text(i18n.T("greeting"))   // "Hello" or "Hola" based on locale
func T(key string) string {
	if instance == nil {
		return key
	}
	// Read current locale reactively (subscribes the active effect)
	locale := instance.current.Val()

	instance.mu.RLock()
	defer instance.mu.RUnlock()

	if trans, ok := instance.locales[locale]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	// Fall back to default locale
	if trans, ok := instance.locales[instance.fallback]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	// Last resort — return the key itself
	return key
}

// TF translates a key and substitutes {placeholders}.
//
//	i18n.Add("en", i18n.Translations{"welcome": "Welcome, {name}!"})
//	i18n.TF("welcome", map[string]string{"name": "Ahmad"})  // "Welcome, Ahmad!"
func TF(key string, vars map[string]string) string {
	result := T(key)
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// AvailableLocales returns all registered locale codes.
func AvailableLocales() []string {
	if instance == nil {
		return nil
	}
	instance.mu.RLock()
	defer instance.mu.RUnlock()
	locales := make([]string, 0, len(instance.locales))
	for code := range instance.locales {
		locales = append(locales, code)
	}
	return locales
}
