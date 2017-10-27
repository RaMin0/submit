$(document).on('click', '#toggle-schedule', function(e) {
  e.preventDefault();
  $('#slot').toggleClass('hidden');
  $('#schedule').toggleClass('hidden');
});
