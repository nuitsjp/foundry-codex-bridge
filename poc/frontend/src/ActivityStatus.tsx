type ActivityStatusProps = {
  message: string;
};

export default function ActivityStatus({ message }: ActivityStatusProps) {
  return (
    <span className="activity-status" role="status" aria-live="polite" aria-busy={Boolean(message)}>
      {message && <><span className="activity-spinner" aria-hidden="true" /><span className="activity-label">{message}</span></>}
    </span>
  );
}
