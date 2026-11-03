function handle(msg, view) {
  switch (msg.type) {
    case 'next': view.next(); break;
    case 'prev': view.prev(); break;
    case 'goTo': if (msg.cfi) view.goTo(msg.cfi); break;
    case 'setStyle': view.setStyle(msg.style); break;
  }
}
module.exports = { handle };
