package io.github.nimbleflux.fluxbase.core

import java.security.SecureRandom
import java.security.cert.X509Certificate
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLSocketFactory
import javax.net.ssl.TrustManager
import javax.net.ssl.X509TrustManager

/**
 * Trust-all TLS material for [KtorHttpTransport] and the realtime WebSocket
 * transport when the caller opts in via
 * `FluxbaseClientOptions.trustSslCertificates` (self-signed self-hosted
 * instances). Scoped to the transports' own OkHttp engines — nothing global
 * is modified.
 */
internal object TrustAllCertificates {

    val trustManager: X509TrustManager = object : X509TrustManager {
        override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) = Unit
        override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) = Unit
        override fun getAcceptedIssuers(): Array<X509Certificate> = arrayOf()
    }

    val socketFactory: SSLSocketFactory = SSLContext.getInstance("TLS").apply {
        init(null, arrayOf<TrustManager>(trustManager), SecureRandom())
    }.socketFactory

    val hostnameVerifier = javax.net.ssl.HostnameVerifier { _, _ -> true }
}
