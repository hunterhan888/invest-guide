import { useTranslation } from 'react-i18next';
import { CompassLogo } from '@/theme/logo';
import { HomeComposer } from '@/features/home/HomeComposer';
import styles from './HomePage.module.css';

export default function HomePage() {
  const { t } = useTranslation();
  return (
    <div className={styles.root}>
      <div className={styles.hero}>
        <div className={styles.logo}>
          <CompassLogo size={52} />
        </div>
        <h1 className={styles.title}>{t('home.welcome')}</h1>
        <p className={styles.subtitle}>{t('home.subtitle')}</p>
        <p className={styles.tagline}>{t('home.tagline')}</p>
      </div>
      <div className={styles.composer}>
        <HomeComposer />
      </div>
    </div>
  );
}
