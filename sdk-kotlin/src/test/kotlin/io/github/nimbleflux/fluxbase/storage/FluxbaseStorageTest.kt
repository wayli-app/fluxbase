package io.github.nimbleflux.fluxbase.storage

import io.github.nimbleflux.fluxbase.FluxbaseClient
import io.github.nimbleflux.fluxbase.FluxbaseClientOptions
import io.github.nimbleflux.fluxbase.core.test.RecordingHttp
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class FluxbaseStorageTest {

    private fun client(recording: RecordingHttp): FluxbaseClient =
        FluxbaseClient.create("http://localhost:8080", "anon-key", FluxbaseClientOptions(autoRefresh = false), recording)

    @Test
    fun `upload posts to storage bucket path`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"path":"photo.jpg","full_path":"trip-images/photo.jpg"}""")
        val c = client(recording)

        c.storage.from("trip-images").upload("photo.jpg", byteArrayOf(1, 2, 3), contentType = "image/jpeg")

        assertEquals("POST", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/storage/trip-images/photo.jpg"))
    }

    @Test
    fun `upload returns result`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"path":"photo.jpg","full_path":"trip-images/photo.jpg"}""")
        val c = client(recording)

        val result = c.storage.from("trip-images").upload("photo.jpg", byteArrayOf(1, 2, 3))

        assertNull(result.error)
        assertEquals("photo.jpg", result.data?.path)
        assertEquals("trip-images/photo.jpg", result.data?.fullPath)
    }

    @Test
    fun `list gets files from bucket`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """[{"name":"a.jpg","size":1024},{"name":"b.jpg","size":2048}]""")
        val c = client(recording)

        val result = c.storage.from("trip-images").list()

        assertEquals("GET", recording.lastMethod)
        assertNull(result.error)
        assertEquals(2, result.data!!.size)
        assertEquals("a.jpg", result.data!![0].name)
    }

    @Test
    fun `download gets file`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "file-content")
        val c = client(recording)

        val result = c.storage.from("trip-images").download("photo.jpg")

        assertEquals("GET", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/storage/trip-images/photo.jpg"))
        assertNull(result.error)
    }

    @Test
    fun `download preserves binary bytes that are not valid utf-8`() = runTest {
        // JPEG SOI (FF D8) + APP0/JFIF marker (FF E0 00 10 4A 46 49 46) — invalid
        // UTF-8 that a String round-trip (bodyAsText → toByteArray) would corrupt.
        // Proves download uses the binary-safe transport path (requestBytes).
        val imageBytes = byteArrayOf(
            0xFF.toByte(), 0xD8.toByte(), 0xFF.toByte(), 0xE0.toByte(),
            0x00, 0x10, 0x4A, 0x46, 0x49, 0x46,
        )
        val recording = RecordingHttp(mockResponseBytes = imageBytes)
        val c = client(recording)

        val result = c.storage.from("trip-images").download("avatar.jpg")

        assertEquals("GET", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/storage/trip-images/avatar.jpg"))
        assertNull(result.error)
        assertTrue(
            result.data!!.contentEquals(imageBytes),
            "downloaded bytes must match the original binary exactly (no charset decode)",
        )
    }

    @Test
    fun `createSignedUrl posts to sign endpoint`() = runTest {
        val recording = RecordingHttp(mockResponseBody = """{"signed_url":"https://example.com/signed"}""")
        val c = client(recording)

        val result = c.storage.from("trip-images").createSignedUrl("photo.jpg", expiresIn = 7200)

        assertEquals("POST", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/sign/photo.jpg"))
        assertNull(result.error)
        assertEquals("https://example.com/signed", result.data?.signedUrl)
    }

    @Test
    fun `remove deletes file`() = runTest {
        val recording = RecordingHttp(mockResponseBody = "[]")
        val c = client(recording)

        c.storage.from("trip-images").remove("photo.jpg")

        assertEquals("DELETE", recording.lastMethod)
        assertTrue(recording.lastPath!!.contains("/api/v1/storage/trip-images/photo.jpg"))
    }

    @Test
    fun `getPublicUrl constructs URL`() {
        val recording = RecordingHttp()
        val c = client(recording)

        val url = c.storage.from("trip-images").getPublicUrl("photo.jpg")

        assertEquals("http://localhost:8080/api/v1/storage/trip-images/photo.jpg", url)
    }
}
