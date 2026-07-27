/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/**
 * Resolve the about content for the active interface language.
 *
 * The backend stores a single `About` option string. Besides the legacy plain
 * HTML/Markdown/URL formats, it may hold a JSON object mapping interface
 * language codes (`zhCN`, `en`, `fr`, `ru`, `ja`, `vi`, `zhTW`) to per-language
 * content, e.g. {"zhCN": "<html...>", "en": "<html...>"}. Falls back to
 * English, then zhCN, then the first string value; non-JSON content is
 * returned unchanged so existing single-language setups keep working.
 */
export function resolveLocalizedAboutContent(
  raw: string,
  language: string
): string {
  const trimmed = raw.trim()
  if (!trimmed.startsWith('{')) return raw
  try {
    const parsed: unknown = JSON.parse(trimmed)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const map = parsed as Record<string, unknown>
      const candidate =
        map[language] ??
        map['en'] ??
        map['zhCN'] ??
        Object.values(map).find((v) => typeof v === 'string')
      if (typeof candidate === 'string' && candidate.trim().length > 0) {
        return candidate
      }
    }
  } catch {
    // Not a JSON language map — treat as plain content.
  }
  return raw
}
