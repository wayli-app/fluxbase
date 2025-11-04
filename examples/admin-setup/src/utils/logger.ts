/**
 * Simple console logger with colors and formatting
 */

export const logger = {
  /**
   * Log success message
   */
  success(message: string, ...args: any[]) {
    console.log(`✅ ${message}`, ...args)
  },

  /**
   * Log error message
   */
  error(message: string, error?: any) {
    console.error(`❌ ${message}`)
    if (error) {
      if (error.message) {
        console.error(`   ${error.message}`)
      }
      if (error.status) {
        console.error(`   Status: ${error.status}`)
      }
    }
  },

  /**
   * Log info message
   */
  info(message: string, ...args: any[]) {
    console.log(`ℹ️  ${message}`, ...args)
  },

  /**
   * Log warning message
   */
  warn(message: string, ...args: any[]) {
    console.warn(`⚠️  ${message}`, ...args)
  },

  /**
   * Log section header
   */
  section(title: string) {
    console.log(`\n${'='.repeat(60)}`)
    console.log(`  ${title}`)
    console.log(`${'='.repeat(60)}\n`)
  },

  /**
   * Log step
   */
  step(step: number, message: string) {
    console.log(`\n${step}. ${message}`)
    console.log(`${'-'.repeat(50)}`)
  },

  /**
   * Log data object
   */
  data(label: string, data: any) {
    console.log(`   ${label}:`, JSON.stringify(data, null, 2))
  },

  /**
   * Log list item
   */
  item(message: string) {
    console.log(`   • ${message}`)
  }
}
