/**
 * Integration Tests for Admin Setup Examples
 *
 * These tests verify that all admin example scripts work correctly
 * against a real Fluxbase instance.
 *
 * Prerequisites:
 * - Fluxbase server running on localhost:8080
 * - Database is reset before tests
 * - Admin credentials configured in .env
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { createClient, type FluxbaseClient } from '@fluxbase/sdk'
import * as dotenv from 'dotenv'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import * as fs from 'fs'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

// Load environment variables
dotenv.config({ path: join(__dirname, '../.env') })

describe('Admin Setup Integration Tests', () => {
  let client: FluxbaseClient
  let adminEmail: string
  let adminPassword: string
  let testUserEmail: string

  beforeAll(async () => {
    // Setup test environment
    adminEmail = process.env.ADMIN_EMAIL || 'admin@fluxbase.local'
    adminPassword = process.env.ADMIN_PASSWORD || 'password'
    testUserEmail = `test-${Date.now()}@example.com`

    client = createClient({
      url: process.env.FLUXBASE_BASE_URL || 'http://localhost:8080'
    })

    // Ensure admin is logged in for tests
    try {
      await client.admin.login({ email: adminEmail, password: adminPassword })
    } catch (error) {
      console.error('Failed to login as admin:', error)
      throw new Error('Cannot run integration tests without admin access')
    }
  })

  afterAll(async () => {
    // Cleanup - logout
    try {
      await client.admin.logout()
    } catch (error) {
      // Ignore logout errors
    }
  })

  describe('01: Admin Authentication', () => {
    it('should authenticate admin successfully', async () => {
      const response = await client.admin.login({
        email: adminEmail,
        password: adminPassword
      })

      expect(response).toBeDefined()
      expect(response.token).toBeDefined()
      expect(response.user).toBeDefined()
      expect(response.user.email).toBe(adminEmail)
      expect(response.user.role).toBe('admin')
    })

    it('should get admin info', async () => {
      const { user } = await client.admin.me()

      expect(user).toBeDefined()
      expect(user.email).toBe(adminEmail)
      expect(user.role).toBe('admin')
    })

    it('should reject invalid credentials', async () => {
      await expect(
        client.admin.login({
          email: adminEmail,
          password: 'wrong-password'
        })
      ).rejects.toThrow()
    })

    it('should logout successfully', async () => {
      await client.admin.logout()

      // After logout, me() should fail
      await expect(client.admin.me()).rejects.toThrow()

      // Re-login for other tests
      await client.admin.login({ email: adminEmail, password: adminPassword })
    })
  })

  describe('02: User Management', () => {
    let createdUserId: string

    it('should list users', async () => {
      const { users, total } = await client.admin.listUsers({ limit: 10 })

      expect(users).toBeDefined()
      expect(Array.isArray(users)).toBe(true)
      expect(typeof total).toBe('number')
    })

    it('should invite a new user', async () => {
      const response = await client.admin.inviteUser({
        email: testUserEmail,
        role: 'user'
      })

      expect(response).toBeDefined()
      expect(response.user).toBeDefined()
      expect(response.user.email).toBe(testUserEmail)
      createdUserId = response.user.id
    })

    it('should get user by ID', async () => {
      const user = await client.admin.getUser(createdUserId)

      expect(user).toBeDefined()
      expect(user.id).toBe(createdUserId)
      expect(user.email).toBe(testUserEmail)
    })

    it('should update user role', async () => {
      const updatedUser = await client.admin.updateUserRole(createdUserId, 'admin')

      expect(updatedUser).toBeDefined()
      expect(updatedUser.role).toBe('admin')

      // Change back to user
      await client.admin.updateUserRole(createdUserId, 'user')
    })

    it('should search users by email', async () => {
      const { users } = await client.admin.listUsers({
        email: testUserEmail
      })

      expect(users.length).toBeGreaterThan(0)
      expect(users[0].email).toBe(testUserEmail)
    })

    it('should reset user password', async () => {
      const response = await client.admin.resetUserPassword(createdUserId)

      expect(response).toBeDefined()
      expect(response.message).toBeDefined()
    })

    it('should delete user', async () => {
      const response = await client.admin.deleteUser(createdUserId)

      expect(response).toBeDefined()

      // Verify user is deleted
      await expect(client.admin.getUser(createdUserId)).rejects.toThrow()
    })
  })

  describe('03: OAuth Configuration', () => {
    it('should list OAuth providers', async () => {
      const { providers } = await client.admin.oauth.providers.list()

      expect(providers).toBeDefined()
      expect(Array.isArray(providers)).toBe(true)
    })

    it('should get auth settings', async () => {
      const settings = await client.admin.oauth.authSettings.get()

      expect(settings).toBeDefined()
      expect(typeof settings.password_min_length).toBe('number')
    })

    it('should update auth settings', async () => {
      const currentSettings = await client.admin.oauth.authSettings.get()

      const updatedSettings = await client.admin.oauth.authSettings.update({
        password_min_length: 12
      })

      expect(updatedSettings.password_min_length).toBe(12)

      // Restore original
      await client.admin.oauth.authSettings.update({
        password_min_length: currentSettings.password_min_length
      })
    })
  })

  describe('04: Settings Management', () => {
    const testSettingKey = `test.setting.${Date.now()}`

    it('should list system settings', async () => {
      const { settings } = await client.admin.settings.system.list()

      expect(settings).toBeDefined()
      expect(Array.isArray(settings)).toBe(true)
    })

    it('should create/update system setting', async () => {
      await client.admin.settings.system.update(testSettingKey, {
        value: { enabled: true, count: 42 },
        description: 'Test setting'
      })

      const setting = await client.admin.settings.system.get(testSettingKey)
      expect(setting).toBeDefined()
      expect(setting.value).toEqual({ enabled: true, count: 42 })
    })

    it('should get specific setting', async () => {
      const setting = await client.admin.settings.system.get(testSettingKey)

      expect(setting).toBeDefined()
      expect(setting.key).toBe(testSettingKey)
      expect(setting.value).toEqual({ enabled: true, count: 42 })
    })

    it('should delete system setting', async () => {
      await client.admin.settings.system.delete(testSettingKey)

      await expect(
        client.admin.settings.system.get(testSettingKey)
      ).rejects.toThrow()
    })

    it('should get app settings', async () => {
      const settings = await client.admin.settings.app.get()

      expect(settings).toBeDefined()
      expect(settings.features).toBeDefined()
    })

    it('should update app settings', async () => {
      const currentSettings = await client.admin.settings.app.get()

      const updated = await client.admin.settings.app.update({
        features: {
          ...currentSettings.features,
          enable_realtime: true
        }
      })

      expect(updated.features?.enable_realtime).toBe(true)
    })
  })

  describe('05: DDL Operations', () => {
    const testSchemaName = `test_schema_${Date.now()}`
    const testTableName = 'test_table'

    it('should list schemas', async () => {
      const schemas = await client.admin.ddl.listSchemas()

      expect(schemas).toBeDefined()
      expect(Array.isArray(schemas)).toBe(true)
      expect(schemas.length).toBeGreaterThan(0)
    })

    it('should create schema', async () => {
      await client.admin.ddl.createSchema(testSchemaName)

      const schemas = await client.admin.ddl.listSchemas()
      const schemaExists = schemas.some(s => s.schema_name === testSchemaName)
      expect(schemaExists).toBe(true)
    })

    it('should create table in schema', async () => {
      await client.admin.ddl.createTable(testSchemaName, testTableName, [
        { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
        { name: 'name', type: 'TEXT', nullable: false },
        { name: 'value', type: 'INTEGER' },
        { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
      ])

      const tables = await client.admin.ddl.listTables(testSchemaName)
      const tableExists = tables.some(t => t.table_name === testTableName)
      expect(tableExists).toBe(true)
    })

    it('should list tables in schema', async () => {
      const tables = await client.admin.ddl.listTables(testSchemaName)

      expect(tables).toBeDefined()
      expect(Array.isArray(tables)).toBe(true)
      expect(tables.length).toBeGreaterThan(0)
    })

    // Cleanup
    it('should drop test schema', async () => {
      // Note: May need cascade option in future
      await client.admin.ddl.dropSchema(testSchemaName)

      const schemas = await client.admin.ddl.listSchemas()
      const schemaExists = schemas.some(s => s.schema_name === testSchemaName)
      expect(schemaExists).toBe(false)
    })
  })

  describe('06: Impersonation', () => {
    let testUserId: string

    beforeAll(async () => {
      // Create a test user for impersonation
      const response = await client.admin.inviteUser({
        email: `impersonation-test-${Date.now()}@example.com`,
        role: 'user'
      })
      testUserId = response.user.id
    })

    afterAll(async () => {
      // Cleanup test user
      try {
        await client.admin.deleteUser(testUserId)
      } catch (error) {
        // Ignore cleanup errors
      }
    })

    it('should check current impersonation status', async () => {
      const current = await client.admin.impersonation.getCurrent()

      expect(current).toBeDefined()
      // Should not be impersonating initially
      expect(current.session).toBeNull()
    })

    it('should impersonate user', async () => {
      const impersonation = await client.admin.impersonation.impersonateUser({
        target_user_id: testUserId,
        reason: 'Integration test - testing user permissions'
      })

      expect(impersonation).toBeDefined()
      expect(impersonation.session).toBeDefined()
      expect(impersonation.session.target_user_id).toBe(testUserId)
      expect(impersonation.target_user).toBeDefined()

      // Verify impersonation is active
      const current = await client.admin.impersonation.getCurrent()
      expect(current.session).toBeDefined()
    })

    it('should stop impersonation', async () => {
      await client.admin.impersonation.stop()

      const current = await client.admin.impersonation.getCurrent()
      expect(current.session).toBeNull()
    })

    it('should impersonate anonymous user', async () => {
      const impersonation = await client.admin.impersonation.impersonateAnon({
        reason: 'Integration test - testing public access'
      })

      expect(impersonation).toBeDefined()
      expect(impersonation.session).toBeDefined()
      expect(impersonation.session.impersonation_type).toBe('anon')

      await client.admin.impersonation.stop()
    })

    it('should impersonate with service role', async () => {
      const impersonation = await client.admin.impersonation.impersonateService({
        reason: 'Integration test - testing service role'
      })

      expect(impersonation).toBeDefined()
      expect(impersonation.session).toBeDefined()
      expect(impersonation.session.impersonation_type).toBe('service')

      await client.admin.impersonation.stop()
    })

    it('should list impersonation sessions', async () => {
      const { sessions, total } = await client.admin.impersonation.listSessions({
        limit: 10
      })

      expect(sessions).toBeDefined()
      expect(Array.isArray(sessions)).toBe(true)
      expect(typeof total).toBe('number')
      // Should have sessions from previous tests
      expect(sessions.length).toBeGreaterThan(0)
    })

    it('should filter sessions by type', async () => {
      const { sessions } = await client.admin.impersonation.listSessions({
        impersonation_type: 'user',
        limit: 10
      })

      expect(sessions).toBeDefined()
      sessions.forEach(session => {
        expect(session.impersonation_type).toBe('user')
      })
    })
  })

  describe('Complete Workflow', () => {
    it('should execute a complete admin workflow', async () => {
      // 1. Login
      await client.admin.login({ email: adminEmail, password: adminPassword })

      // 2. Create schema
      const schemaName = `workflow_${Date.now()}`
      await client.admin.ddl.createSchema(schemaName)

      // 3. Create table
      await client.admin.ddl.createTable(schemaName, 'users', [
        { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
        { name: 'email', type: 'CITEXT', nullable: false },
        { name: 'name', type: 'TEXT' }
      ])

      // 4. Create user
      const userEmail = `workflow-${Date.now()}@example.com`
      const { user } = await client.admin.inviteUser({
        email: userEmail,
        role: 'user'
      })

      // 5. Configure settings
      await client.admin.settings.system.update(`workflow.${Date.now()}`, {
        value: { test: true },
        description: 'Workflow test setting'
      })

      // 6. Impersonate user
      await client.admin.impersonation.impersonateUser({
        target_user_id: user.id,
        reason: 'Workflow test'
      })

      // 7. Stop impersonation
      await client.admin.impersonation.stop()

      // 8. Cleanup
      await client.admin.deleteUser(user.id)
      await client.admin.ddl.dropSchema(schemaName)

      // Workflow completed successfully
      expect(true).toBe(true)
    })
  })
})
