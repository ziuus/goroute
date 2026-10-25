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
      <body className="antialiased selection:bg-blue-500/20 selection:text-blue-600">
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
