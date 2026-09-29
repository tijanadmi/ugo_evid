import { useEffect, useId, useRef, useState } from 'react';
import Icon from './Icon';

export default function RowMenu({ label, actions }) {
  const id = useId();
  const toggle = useRef(null);
  const menu = useRef(null);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const hide = () => menu.current?.hidePopover();
    window.addEventListener('resize', hide);
    window.addEventListener('scroll', hide, true);
    return () => {
      window.removeEventListener('resize', hide);
      window.removeEventListener('scroll', hide, true);
    };
  }, [open]);
  function close() {
    menu.current.hidePopover();
    toggle.current.focus();
  }
  function position() {
    const rect = toggle.current.getBoundingClientRect();
    const width = 176;
    const height = actions.length * 42 + 10;
    Object.assign(menu.current.style, {
      left: `${Math.max(8, Math.min(rect.right - width, window.innerWidth - width - 8))}px`,
      top: `${Math.max(8, rect.bottom + height + 8 > window.innerHeight ? rect.top - height - 6 : rect.bottom + 6)}px`,
    });
  }
  return <div className="row-menu">
    <button ref={toggle} type="button" className="icon-button row-menu-toggle" aria-label={label} aria-haspopup="menu" aria-expanded={open} aria-controls={id} popoverTarget={id} onClick={position} onKeyDown={event => {
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        position();
        menu.current.showPopover();
      }
    }}><Icon name="more"/></button>
    <div ref={menu} id={id} popover="auto" role="menu" aria-label={label} className="row-menu-list" onToggle={event => {
      const visible = event.newState === 'open';
      setOpen(visible);
      if (visible) menu.current.querySelector('[role="menuitem"]')?.focus();
    }} onKeyDown={event => {
      if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(); }
      if (event.key === 'Tab') { menu.current.hidePopover(); }
      if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
        event.preventDefault();
        const items = [...menu.current.querySelectorAll('[role="menuitem"]')];
        const current = items.indexOf(document.activeElement);
        const next = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (current + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
        items[next]?.focus();
      }
    }}>
      {actions.map(action => <button key={action.label} type="button" role="menuitem" className={action.danger ? 'danger' : ''} onClick={() => { close(); action.onClick(); }}><Icon name={action.icon}/><span>{action.label}</span></button>)}
    </div>
  </div>;
}
