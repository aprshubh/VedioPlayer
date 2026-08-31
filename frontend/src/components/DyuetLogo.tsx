import React from 'react';

interface DyuetLogoProps {
  size?: number;
  className?: string;
}

export const DyuetLogo: React.FC<DyuetLogoProps> = ({ size = 28, className = '' }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Outer rounded geometric container */}
      <rect width="40" height="40" rx="10" fill="currentColor" fillOpacity="0.08" />
      
      {/* Primary Left Spine of the 'D' */}
      <path
        d="M12 9C10.8954 9 10 9.89543 10 11V29C10 30.1046 10.8954 31 12 31H14C15.1046 31 16 30.1046 16 29V11C16 9.89543 15.1046 9 14 9H12Z"
        fill="currentColor"
      />

      {/* Synchronized Duet Sweeping Arc of the 'D' */}
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M15 9H21C26.5228 9 31 13.4772 31 19C31 24.5228 26.5228 29 21 29H15V24H21C23.7614 24 26 21.7614 26 19C26 16.2386 23.7614 14 21 14H15V9Z"
        fill="currentColor"
      />

      {/* Inner Synchronized Sync Dot / Cine Pulse */}
      <circle cx="21" cy="19" r="2.5" fill="currentColor" />
    </svg>
  );
};
