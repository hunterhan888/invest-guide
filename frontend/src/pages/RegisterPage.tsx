import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Input } from '@/primitives/Input';
import { useToast } from '@/primitives/ToastProvider';
import { CompassLogo } from '@/theme/logo';
import { register as apiRegister } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import { Field } from '@/features/auth/fields';
import styles from './RegisterPage.module.css';

type FormValues = {
  displayName: string;
  email: string;
  password: string;
  confirmPassword: string;
};

export default function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [form, setForm] = useState<FormValues>({
    displayName: '',
    email: '',
    password: '',
    confirmPassword: '',
  });
  const [loading, setLoading] = useState(false);
  const [mismatch, setMismatch] = useState(false);

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  function set<K extends keyof FormValues>(key: K, value: string) {
    setForm((f) => ({ ...f, [key]: value }));
    if (key === 'confirmPassword' || key === 'password') {
      setMismatch(false);
    }
  }

  async function onRegister(e: React.FormEvent) {
    e.preventDefault();
    if (loading) return;
    if (form.password !== form.confirmPassword) {
      setMismatch(true);
      return;
    }
    setLoading(true);
    try {
      const req = {
        email: form.email,
        password: form.password,
        displayName: form.displayName,
      };
      const res = await apiRegister(req);
      loginStore({ token: res.token, user: res.user, remember: true });
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const code = (err as { code?: string })?.code;
      toast.error(code === 'CONFLICT' ? t('auth.error.conflict') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <div className={styles.card}>
        <div className={styles.head}>
          <CompassLogo size={44} />
          <h1 className={styles.title}>{t('auth.register.title')}</h1>
          <p className={styles.subtitle}>{t('auth.register.subtitle')}</p>
        </div>
        <form onSubmit={(e) => void onRegister(e)}>
          <Field label={t('auth.field.displayName')}>
            <Input
              value={form.displayName}
              onChange={(e) => set('displayName', e.target.value)}
              placeholder={t('auth.field.displayName')}
              required
            />
          </Field>
          <Field label={t('auth.field.email')}>
            <Input
              type="email"
              value={form.email}
              onChange={(e) => set('email', e.target.value)}
              placeholder={t('auth.field.email')}
              required
            />
          </Field>
          <Field label={t('auth.field.password')}>
            <Input
              type="password"
              value={form.password}
              onChange={(e) => set('password', e.target.value)}
              placeholder={t('auth.field.password')}
              required
              minLength={8}
              maxLength={72}
            />
          </Field>
          <Field
            label={t('auth.field.confirmPassword')}
            error={mismatch ? t('auth.error.passwordMismatch') : undefined}
          >
            <Input
              type="password"
              value={form.confirmPassword}
              onChange={(e) => set('confirmPassword', e.target.value)}
              placeholder={t('auth.field.confirmPassword')}
              required
            />
          </Field>
          <Button type="submit" variant="primary" block size="md" loading={loading}>
            {t('auth.register.submit')}
          </Button>
        </form>
        <div className={styles.switch}>
          <Button variant="ghost" size="sm" onClick={() => navigate('/login')}>
            {t('auth.register.toLogin')}
          </Button>
        </div>
      </div>
    </div>
  );
}
