'use client';

import React, { useEffect, useState, useRef } from 'react';

export default function CustomCursor() {
  const [position, setPosition] = useState({ x: -100, y: -100 });
  const [trailingPos, setTrailingPos] = useState({ x: -100, y: -100 });
  const [isHovered, setIsHovered] = useState(false);
  const [isClicking, setIsClicking] = useState(false);
  const [isVisible, setIsVisible] = useState(false);
  const [isFinePointer, setIsFinePointer] = useState(false);

  const posRef = useRef({ x: -100, y: -100 });
  const trailingRef = useRef({ x: -100, y: -100 });
  const animFrameRef = useRef<number | null>(null);

  useEffect(() => {
    // Only enable custom cursor for precise pointer devices (desktops/laptops, not touch screens)
    if (typeof window === 'undefined') return;
    const mediaQuery = window.matchMedia('(pointer: fine)');
    setIsFinePointer(mediaQuery.matches);

    const handleMediaChange = (e: MediaQueryListEvent) => {
      setIsFinePointer(e.matches);
    };
    mediaQuery.addEventListener('change', handleMediaChange);

    const handleMouseMove = (e: MouseEvent) => {
      posRef.current = { x: e.clientX, y: e.clientY };
      setPosition({ x: e.clientX, y: e.clientY });
      if (!isVisible) setIsVisible(true);

      // Check if target or its ancestors are interactive
      const target = e.target as HTMLElement | null;
      if (target) {
        const interactive = !!target.closest(
          'button, a, input, select, [role="button"], .liquid-card, .clickable, [data-interactive="true"]'
        );
        setIsHovered(interactive);
      }
    };

    const handleMouseDown = () => setIsClicking(true);
    const handleMouseUp = () => setIsClicking(false);

    const handleMouseLeave = () => setIsVisible(false);
    const handleMouseEnter = () => setIsVisible(true);

    window.addEventListener('mousemove', handleMouseMove, { passive: true });
    window.addEventListener('mousedown', handleMouseDown);
    window.addEventListener('mouseup', handleMouseUp);
    document.addEventListener('mouseleave', handleMouseLeave);
    document.addEventListener('mouseenter', handleMouseEnter);

    // Smooth RAF loop for ring trailing physics
    const loop = () => {
      const ease = 0.18;
      trailingRef.current.x += (posRef.current.x - trailingRef.current.x) * ease;
      trailingRef.current.y += (posRef.current.y - trailingRef.current.y) * ease;
      setTrailingPos({ x: trailingRef.current.x, y: trailingRef.current.y });
      animFrameRef.current = requestAnimationFrame(loop);
    };
    animFrameRef.current = requestAnimationFrame(loop);

    return () => {
      mediaQuery.removeEventListener('change', handleMediaChange);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mousedown', handleMouseDown);
      window.removeEventListener('mouseup', handleMouseUp);
      document.removeEventListener('mouseleave', handleMouseLeave);
      document.removeEventListener('mouseenter', handleMouseEnter);
      if (animFrameRef.current) cancelAnimationFrame(animFrameRef.current);
    };
  }, [isVisible]);

  if (!isFinePointer || !isVisible) return null;

  return (
    <>
      {/* High-precision center dot */}
      <div
        className="pointer-events-none fixed z-[99999] rounded-full transition-transform duration-75 ease-out"
        style={{
          left: `${position.x}px`,
          top: `${position.y}px`,
          width: isHovered ? '4px' : '6px',
          height: isHovered ? '4px' : '6px',
          transform: `translate(-50%, -50%) scale(${isClicking ? 0.7 : 1})`,
          backgroundColor: 'var(--accent, #3b82f6)',
          opacity: 0.9,
        }}
      />

      {/* Smooth fluid trailing aura/ring */}
      <div
        className="pointer-events-none fixed z-[99998] rounded-full transition-[width,height,opacity,background-color] duration-200 ease-out"
        style={{
          left: `${trailingPos.x}px`,
          top: `${trailingPos.y}px`,
          width: isClicking ? '26px' : isHovered ? '46px' : '30px',
          height: isClicking ? '26px' : isHovered ? '46px' : '30px',
          transform: 'translate(-50%, -50%)',
          border: isHovered ? '1.5px solid var(--accent, #3b82f6)' : '1.5px solid rgba(140, 160, 200, 0.4)',
          backgroundColor: isHovered ? 'rgba(59, 130, 246, 0.12)' : 'transparent',
          boxShadow: isHovered ? '0 0 16px rgba(59, 130, 246, 0.2)' : 'none',
        }}
      />
    </>
  );
}
