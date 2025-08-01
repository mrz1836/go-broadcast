// Theme management
function toggleTheme() {
  const html = document.documentElement;
  const currentTheme = html.getAttribute('data-theme');
  const newTheme = currentTheme === 'light' ? 'dark' : 'light';
  html.setAttribute('data-theme', newTheme);
  localStorage.setItem('theme', newTheme);
}

// Initialize theme
const savedTheme = localStorage.getItem('theme');
const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
const theme = savedTheme || (systemPrefersDark ? 'dark' : 'light');
document.documentElement.setAttribute('data-theme', theme);

// Package toggle
function togglePackage(packageName) {
  const packageEl = document.getElementById('package-' + packageName);
  const toggleIcon = document.querySelector('[data-package="' + packageName + '"] .package-toggle');

  if (packageEl.style.display === 'none' || !packageEl.style.display) {
    packageEl.style.display = 'block';
    toggleIcon.textContent = '▼';
  } else {
    packageEl.style.display = 'none';
    toggleIcon.textContent = '▶';
  }
}

// Search functionality
const searchInput = document.getElementById('searchInput');
if (searchInput) {
  searchInput.addEventListener('input', function(e) {
    const searchTerm = e.target.value.toLowerCase();
    const packages = document.querySelectorAll('.package-card');

    packages.forEach(pkg => {
      const packageName = pkg.querySelector('.package-name').textContent.toLowerCase();
      const files = pkg.querySelectorAll('.file-item');
      let hasMatch = packageName.includes(searchTerm);

      files.forEach(file => {
        const fileName = file.querySelector('.file-name').textContent.toLowerCase();
        if (fileName.includes(searchTerm)) {
          hasMatch = true;
          file.style.display = 'flex';
        } else if (searchTerm) {
          file.style.display = 'none';
        } else {
          file.style.display = 'flex';
        }
      });

      pkg.style.display = hasMatch || !searchTerm ? 'block' : 'none';

      // Auto-expand packages with matching files
      if (hasMatch && searchTerm) {
        const filesContainer = pkg.querySelector('.package-files');
        if (filesContainer && filesContainer.style.display === 'none') {
          togglePackage(pkg.dataset.package);
        }
      }
    });
  });
}

// Copy badge URL
function copyBadgeURL(event, url) {
  navigator.clipboard.writeText(url).then(() => {
    const btn = event.target.closest('button');
    const originalText = btn.querySelector('.btn-text').textContent;
    btn.querySelector('.btn-text').textContent = 'Copied!';
    setTimeout(() => {
      btn.querySelector('.btn-text').textContent = originalText;
    }, 2000);
  }).catch(err => {
    console.error('Failed to copy badge URL:', err);
    const btn = event.target.closest('button');
    const originalText = btn.querySelector('.btn-text').textContent;
    btn.querySelector('.btn-text').textContent = 'Copy failed';
    setTimeout(() => {
      btn.querySelector('.btn-text').textContent = originalText;
    }, 2000);
  });
}
