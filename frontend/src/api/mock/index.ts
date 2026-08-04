import { installAuthMocks } from './auth';
import { installConversationMocks } from './conversation';

export function installMocks() {
  installAuthMocks();
  installConversationMocks();
}
