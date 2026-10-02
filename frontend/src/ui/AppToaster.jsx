import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { ToastBar, Toaster } from 'react-hot-toast';

// Native modal dialogs live above the document's z-index layers. Keep the one
// global toaster inside the active dialog so notifications remain visible.
export default function AppToaster() {
  const [target, setTarget] = useState(document.body);
  useEffect(() => {
    const update = () => setTarget([...document.querySelectorAll('dialog[open]')].at(-1) || document.body);
    const observer = new MutationObserver(update);
    observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['open'] });
    update();
    return () => observer.disconnect();
  }, []);
  return createPortal(<Toaster position="top-center" gutter={12} containerStyle={{ margin: '8px' }} toastOptions={{
    success: { duration: 3000 }, error: { duration: 5000, ariaProps: { role: 'alert', 'aria-live': 'assertive' } },
    style: { fontSize: '16px', maxWidth: '500px', padding: '16px 24px', backgroundColor: 'var(--color-grey-0)', color: 'var(--color-grey-700)' },
  }}>{item => <ToastBar toast={{ ...item, ariaProps: item.type === 'error' ? { role: 'alert', 'aria-live': 'assertive' } : item.ariaProps }}/>}</Toaster>, target);
}
