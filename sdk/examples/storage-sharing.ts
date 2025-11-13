/**
 * File Sharing Example
 *
 * This example demonstrates how to use the Storage RLS file sharing features:
 * 1. Upload a file as User 1
 * 2. Share it with User 2 (read permission)
 * 3. User 2 accesses the file
 * 4. Upgrade to write permission
 * 5. User 2 modifies/deletes the file
 * 6. List and revoke shares
 */

import { createClient } from '../src/index'

// Initialize clients for two different users
const user1Client = createClient({
  url: 'http://localhost:3000',
  apiKey: 'user1-token-here' // User 1's auth token
})

const user2Client = createClient({
  url: 'http://localhost:3000',
  apiKey: 'user2-token-here' // User 2's auth token
})

// Admin client (for bucket creation)
const adminClient = createClient({
  url: 'http://localhost:3000',
  serviceKey: 'service-key-here'
})

async function fileSharingExample() {
  const bucketName = 'shared-workspace'
  const fileName = 'collaboration.txt'
  const user2Id = 'user-2-uuid-here' // User 2's UUID

  console.log('=== File Sharing Example ===\n')

  // Step 1: Admin creates a private bucket
  console.log('1. Creating private bucket...')
  const { error: createError } = await adminClient.storage.createBucket(bucketName)
  if (createError) {
    console.error('Failed to create bucket:', createError)
    return
  }
  console.log('✅ Bucket created\n')

  // Step 2: User 1 uploads a file
  console.log('2. User 1 uploading file...')
  const file = new File(['Initial content from User 1'], fileName, {
    type: 'text/plain'
  })

  const { data: uploadData, error: uploadError } = await user1Client
    .storage
    .from(bucketName)
    .upload(fileName, file)

  if (uploadError) {
    console.error('Failed to upload file:', uploadError)
    return
  }
  console.log('✅ File uploaded:', uploadData)
  console.log('')

  // Step 3: User 2 tries to access (should fail)
  console.log('3. User 2 attempting to access (should fail)...')
  const { error: download1Error } = await user2Client
    .storage
    .from(bucketName)
    .download(fileName)

  if (download1Error) {
    console.log('✅ Access denied (as expected):', download1Error.message)
  } else {
    console.log('❌ Unexpected: User 2 accessed file without permission')
  }
  console.log('')

  // Step 4: User 1 shares with User 2 (read permission)
  console.log('4. User 1 sharing file with User 2 (read permission)...')
  const { error: shareError } = await user1Client
    .storage
    .from(bucketName)
    .share(fileName, {
      userId: user2Id,
      permission: 'read'
    })

  if (shareError) {
    console.error('Failed to share file:', shareError)
    return
  }
  console.log('✅ File shared with read permission\n')

  // Step 5: User 2 can now download
  console.log('5. User 2 downloading shared file...')
  const { data: blob, error: download2Error } = await user2Client
    .storage
    .from(bucketName)
    .download(fileName)

  if (download2Error) {
    console.error('Failed to download:', download2Error)
  } else {
    const content = await blob!.text()
    console.log('✅ File downloaded successfully')
    console.log('   Content:', content)
  }
  console.log('')

  // Step 6: User 2 tries to delete (should fail - read-only)
  console.log('6. User 2 trying to delete (should fail - read-only)...')
  const { error: delete1Error } = await user2Client
    .storage
    .from(bucketName)
    .remove([fileName])

  if (delete1Error) {
    console.log('✅ Delete denied (as expected):', delete1Error.message)
  } else {
    console.log('❌ Unexpected: User 2 deleted file with read-only permission')
  }
  console.log('')

  // Step 7: User 1 upgrades to write permission
  console.log('7. User 1 upgrading to write permission...')
  const { error: upgradeError } = await user1Client
    .storage
    .from(bucketName)
    .share(fileName, {
      userId: user2Id,
      permission: 'write'
    })

  if (upgradeError) {
    console.error('Failed to upgrade permission:', upgradeError)
    return
  }
  console.log('✅ Permission upgraded to write\n')

  // Step 8: User 2 can now delete
  console.log('8. User 2 deleting file (should succeed)...')
  const { error: delete2Error } = await user2Client
    .storage
    .from(bucketName)
    .remove([fileName])

  if (delete2Error) {
    console.error('Failed to delete:', delete2Error)
  } else {
    console.log('✅ File deleted successfully')
  }
  console.log('')

  // Step 9: Re-upload and demonstrate listing shares
  console.log('9. Re-uploading file to demonstrate share listing...')
  const file2 = new File(['Content for share listing demo'], fileName, {
    type: 'text/plain'
  })

  await user1Client.storage.from(bucketName).upload(fileName, file2)

  await user1Client.storage.from(bucketName).share(fileName, {
    userId: user2Id,
    permission: 'read'
  })

  const { data: shares, error: listError } = await user1Client
    .storage
    .from(bucketName)
    .listShares(fileName)

  if (listError) {
    console.error('Failed to list shares:', listError)
  } else {
    console.log('✅ File shares:')
    shares?.forEach(share => {
      console.log(`   - User: ${share.user_id}`)
      console.log(`     Permission: ${share.permission}`)
      console.log(`     Shared at: ${share.created_at}`)
    })
  }
  console.log('')

  // Step 10: Revoke share
  console.log('10. User 1 revoking access...')
  const { error: revokeError } = await user1Client
    .storage
    .from(bucketName)
    .revokeShare(fileName, user2Id)

  if (revokeError) {
    console.error('Failed to revoke share:', revokeError)
  } else {
    console.log('✅ Access revoked')
  }

  // Verify revocation
  const { error: download3Error } = await user2Client
    .storage
    .from(bucketName)
    .download(fileName)

  if (download3Error) {
    console.log('✅ Access denied after revocation (as expected)')
  } else {
    console.log('❌ Unexpected: User 2 still has access after revocation')
  }
  console.log('')

  console.log('=== Example Complete ===')
}

// Run the example
fileSharingExample().catch(console.error)
