// Indirizzo del backend Go raggiungibile dal server di Next.js.
const backendURL = process.env.BACKEND_URL ?? "http://localhost:8080";

/** @type {import('next').NextConfig} */
const nextConfig = {
  // Non dichiarare la tecnologia usata nell'header X-Powered-By.
  poweredByHeader: false,

  // In sviluppo il browser parla solo con Next.js (porta 3000), che inoltra
  // /api al backend: frontend e API restano sulla stessa origine, quindi
  // niente CORS e i cookie SameSite=Strict vengono inviati. In Docker la
  // stessa cosa la fa nginx, e le richieste /api non arrivano a Next.js.
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${backendURL}/api/:path*` }];
  },
};

export default nextConfig;
