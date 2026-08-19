import { useTranslation } from 'react-i18next';
import styles from './TurnStatus.module.css';

export function TurnStatus() {
  const { t } = useTranslation();
  return <div className={styles.status}>{t('message.streaming.pending')}</div>;
}
