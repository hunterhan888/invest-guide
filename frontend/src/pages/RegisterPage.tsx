import { useEffect, useState } from 'react';
import { App as AntdApp, Button, Form, Input } from 'antd';
import { CompassOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { register as apiRegister } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import type { RegisterRequest } from '@/api/auth/types';

type RegisterFormValues = {
  email: string;
  password: string;
  displayName: string;
  confirmPassword: string;
};

export default function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = AntdApp.useApp();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<RegisterFormValues>();

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  async function onRegister(values: RegisterFormValues) {
    setLoading(true);
    try {
      const req: RegisterRequest = {
        email: values.email,
        password: values.password,
        displayName: values.displayName,
      };
      const res = await apiRegister(req);
      loginStore({ token: res.token, user: res.user, remember: true });
      navigate('/', { replace: true });
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      message.error(code === 'CONFLICT' ? t('auth.error.conflict') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-layout px-4">
      <div className="w-full max-w-[360px]">
        <div className="mb-8 text-center">
          <div
            className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl"
            style={{ background: 'linear-gradient(135deg, #4096ff, #1677ff)' }}
          >
            <CompassOutlined style={{ fontSize: 26, color: '#fff' }} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-fg">{t('auth.register.title')}</h1>
          <p className="mt-1 text-fg-secondary">{t('auth.register.subtitle')}</p>
        </div>
        <Form form={form} layout="vertical" onFinish={onRegister}>
          <Form.Item name="displayName" rules={[{ required: true }]}>
            <Input placeholder={t('auth.field.displayName')} size="large" />
          </Form.Item>
          <Form.Item name="email" rules={[{ required: true }, { type: 'email' }]}>
            <Input
              prefix={<MailOutlined className="text-fg-tertiary" />}
              placeholder={t('auth.field.email')}
              size="large"
            />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true }, { min: 8 }, { max: 72 }]}>
            <Input.Password
              prefix={<LockOutlined className="text-fg-tertiary" />}
              placeholder={t('auth.field.password')}
              size="large"
            />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            dependencies={['password']}
            rules={[
              { required: true },
              ({ getFieldValue }) => ({
                validator(_, value: string) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(t('auth.error.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined className="text-fg-tertiary" />}
              placeholder={t('auth.field.confirmPassword')}
              size="large"
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large" loading={loading}>
            {t('auth.register.submit')}
          </Button>
        </Form>
        <div className="mt-4 text-center">
          <Button type="link" onClick={() => navigate('/login')}>
            {t('auth.register.toLogin')}
          </Button>
        </div>
      </div>
    </div>
  );
}
