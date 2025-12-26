// Lightweight collapsible rows for transfer templates
(function() {
  'use strict';

  function toggleChildren(parentRow) {
    const templateId = parentRow.getAttribute('data-template-id');
    if (!templateId) return;

    const childRows = document.querySelectorAll('tr[data-parent-id="' + templateId + '"]');
    if (childRows.length === 0) return;

    const isCollapsed = parentRow.classList.contains('collapsed');

    if (isCollapsed) {
      parentRow.classList.remove('collapsed');
      for (let i = 0; i < childRows.length; i++) {
        childRows[i].classList.remove('hidden');
      }
    } else {
      parentRow.classList.add('collapsed');
      for (let i = 0; i < childRows.length; i++) {
        childRows[i].classList.add('hidden');
      }
    }
  }

  // Expose function globally for inline onclick handlers
  window.toggleTemplateChildren = function(button) {
    const parentRow = button.closest('tr[data-template-id]');
    if (parentRow) {
      toggleChildren(parentRow);
    }
  };

  function initCollapsibleRows() {
    // Find all parent rows
    const parentRows = document.querySelectorAll('tr[data-template-id]');
    
    for (let i = 0; i < parentRows.length; i++) {
      const parentRow = parentRows[i];
      const templateId = parentRow.getAttribute('data-template-id');
      if (!templateId) continue;
      
      const childRows = document.querySelectorAll('tr[data-parent-id="' + templateId + '"]');
      
      // Set initial state - collapsed by default
      if (childRows.length > 0) {
        parentRow.classList.add('collapsed');
        for (let j = 0; j < childRows.length; j++) {
          childRows[j].classList.add('hidden');
        }
      }
    }
  }

  // Initialize immediately and on DOM ready
  initCollapsibleRows();
  
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initCollapsibleRows);
  }
})();

