import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2, NavModelItem } from '@grafana/data';
import { IconName, Link, useTheme2 } from '@grafana/ui';
import DropdownChild from './DropdownChild';

interface Props {
  headerTarget?: HTMLAnchorElement['target'];
  headerText: string;
  headerUrl?: string;
  items?: NavModelItem[];
  onHeaderClick?: () => void;
  reverseDirection?: boolean;
  subtitleText?: string;
}

const NavBarDropdown = ({
  headerTarget,
  headerText,
  headerUrl,
  items = [],
  onHeaderClick,
  reverseDirection = false,
  subtitleText,
}: Props) => {
  const filteredItems = items.filter((item) => !item.hideFromMenu);
  const theme = useTheme2();
  const styles = getStyles(theme, reverseDirection, filteredItems);

  let header = (
    <button onClick={onHeaderClick} className={styles.header}>
      {headerText}
    </button>
  );
  if (headerUrl) {
    header =
      !headerTarget && headerUrl.startsWith('/') ? (
        <Link href={headerUrl} onClick={onHeaderClick} className={styles.header}>
          {headerText}
        </Link>
      ) : (
        <a href={headerUrl} target={headerTarget} onClick={onHeaderClick} className={styles.header}>
          {headerText}
        </a>
      );
  }

  return (
    <ul className={`${styles.menu} dropdown-menu dropdown-menu--sidemenu`} role="menu">
      <li>{header}</li>
      {filteredItems.map((child, index) => (
        <DropdownChild
          key={`${child.url}-${index}`}
          isDivider={child.divider}
          icon={child.icon as IconName}
          onClick={child.onClick}
          target={child.target}
          text={child.text}
          url={child.url}
        />
      ))}
      {subtitleText && <li className={styles.subtitle}>{subtitleText}</li>}
    </ul>
  );
};

export default NavBarDropdown;

const getStyles = (
  theme: GrafanaTheme2,
  reverseDirection: Props['reverseDirection'],
  filteredItems: Props['items']
) => {
  const adjustHeightForBorder = filteredItems!.length === 0;

  return {
    header: css`
      align-items: center;
      background-color: ${theme.colors.background.secondary};
      border: none;
      color: ${theme.colors.text.primary};
      height: ${theme.components.sidemenu.width - (adjustHeightForBorder ? 2 : 1)}px;
      font-size: ${theme.typography.h4.fontSize};
      font-weight: ${theme.typography.h4.fontWeight};
      padding: ${theme.spacing(1)} ${theme.spacing(1)} ${theme.spacing(1)} ${theme.spacing(2)} !important;
      white-space: nowrap;
      width: 100%;

      &:hover {
        background-color: ${theme.colors.action.hover};
      }

      .sidemenu-open--xs & {
        display: flex;
        font-size: ${theme.typography.body.fontSize};
        font-weight: ${theme.typography.body.fontWeight};
        padding-left: ${theme.spacing(1)} !important;
      }
    `,
    menu: css`
      border: 1px solid ${theme.components.panel.borderColor};
      flex-direction: ${reverseDirection ? 'column-reverse' : 'column'};

      .sidemenu-open--xs & {
        display: flex;
        flex-direction: column;
        float: none;
        position: unset;
        width: 100%;
      }
    `,
    subtitle: css`
      border-${reverseDirection ? 'bottom' : 'top'}: 1px solid ${theme.colors.border.weak};
      color: ${theme.colors.text.secondary};
      font-size: ${theme.typography.bodySmall.fontSize};
      font-weight: ${theme.typography.bodySmall.fontWeight};
      padding: ${theme.spacing(1)} ${theme.spacing(2)} ${theme.spacing(1)};
      white-space: nowrap;

      .sidemenu-open--xs & {
        border-${reverseDirection ? 'bottom' : 'top'}: none;
      }
    `,
  };
};
