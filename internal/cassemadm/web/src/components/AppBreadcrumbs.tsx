import { Breadcrumbs, Link, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'

type BreadcrumbItem = {
  label: string
  to?: string
}

type AppBreadcrumbsProps = {
  items: BreadcrumbItem[]
}

export function AppBreadcrumbs({ items }: AppBreadcrumbsProps) {
  return (
    <Breadcrumbs aria-label="breadcrumb">
      {items.map((item, index) => {
        const to = item.to
        if (index === items.length - 1 || !to) {
          return (
            <Typography key={`${item.label}-${index}`} color="text.primary" sx={{ overflowWrap: 'anywhere' }}>
              {item.label}
            </Typography>
          )
        }

        return (
          <Link
            key={`${item.label}-${index}`}
            component={RouterLink}
            to={to}
            underline="always"
            color="primary"
            sx={{ fontWeight: 600, overflowWrap: 'anywhere', textUnderlineOffset: 3 }}
          >
            {item.label}
          </Link>
        )
      })}
    </Breadcrumbs>
  )
}
