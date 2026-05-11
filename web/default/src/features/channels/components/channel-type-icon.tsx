/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { SVGProps } from 'react'
import { Server } from 'lucide-react'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { GLOBAL_PASSTHROUGH_TYPE } from '../constants'
import { getChannelTypeIcon } from '../lib/channel-utils'

type IconProps = SVGProps<SVGSVGElement> & {
  size?: number
}

export function GlobalPassthroughIcon({
  size = 20,
  className,
  ...props
}: IconProps) {
  return (
    <svg
      xmlns='http://www.w3.org/2000/svg'
      width={size}
      height={size}
      viewBox='0 0 24 24'
      fill='none'
      stroke='currentColor'
      strokeWidth={2}
      strokeLinecap='round'
      strokeLinejoin='round'
      className={cn('text-cyan-500', className)}
      aria-hidden='true'
      {...props}
    >
      <circle cx='6' cy='19' r='3' />
      <path d='M9 19h8.5a3.5 3.5 0 0 0 0-7H6.5a3.5 3.5 0 0 1 0-7H15' />
      <circle cx='18' cy='5' r='3' />
    </svg>
  )
}

export function ChannelTypeIcon({
  type,
  size = 20,
  className,
}: {
  type: number
  size?: number
  className?: string
}) {
  if (type === GLOBAL_PASSTHROUGH_TYPE) {
    return <GlobalPassthroughIcon size={size} className={className} />
  }

  if (!Number.isInteger(type) || type <= 0) {
    return (
      <Server
        size={size}
        className={cn('text-muted-foreground', className)}
      />
    )
  }

  return getLobeIcon(`${getChannelTypeIcon(type)}.Color`, size)
}
