import { Chip, Tooltip } from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ErrorOutlinedIcon from '@mui/icons-material/ErrorOutlined';
import type { StatusCustomer } from '@/lib/tracking';

// Same color language as the customer timeline: bad = error, good with warnings = warning, else primary.
export default function StatusChip({ status }: { status: StatusCustomer }) {
  switch (status.statusType) {
    case 'bad':
      return (
        <Chip size="small" color="error" icon={<ErrorOutlinedIcon />} label={status.status} />
      );
    case 'good':
      if (status.warnings.length === 0) {
        return <Chip size="small" color="primary" variant="outlined" label={status.status} />;
      }
      return (
        <Tooltip title={status.warnings.join(' • ')}>
          <Chip size="small" color="warning" icon={<WarningAmberIcon />} label={status.status} />
        </Tooltip>
      );
  }
}
