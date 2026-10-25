import type { Metadata } from 'next';
import './globals.css';
import CustomCursor from './CustomCursor';

export const metadata: Metadata = {
  title: 'GoRoute — Universal LLM Gateway (Go + Rust)',
  description: 'Ultra-fast, high-throughput AI gateway and multi-provider router (< 15 MB RAM)',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Quicksand:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="antialiased selection:bg-rose-500/20 selection:text-rose-600">
        <CustomCursor />
        {/* Layer 1: Animated Aurora Gradient Mesh */}
        <div className="aurora-mesh" />
        {/* Layer 2: Subtle Film Grain Noise Texture */}
        <div className="noise-overlay" />
        {/* Layer 3: Interactive UI */}
        <div className="relative z-10 flex min-h-screen flex-col">
          {children}
        </div>
      </body>
    </html>
  );
}
