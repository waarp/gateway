.. _changelog:

Historique des versions
=======================

* :bug:`153` La mise-à-jour partielle de la base de données via la commande
  ``import`` n'est plus autorisée. Les objets doivent désormais être renseignés
  en intégralité dans le fichier importé pour que l'opération puisse se faire.
* :feature:`153` Le paramètre ``--config`` (ou ``-c``) des commandes ``server add``
  et ``partner add`` du client est désormais obligatoire.
* :feature:`153` Dans l'API REST, le champ ``paths`` de l'objet serveur a été
  supprimé. À la place, les différents chemins contenus dans ``paths`` ont été
  ramenés directement dans l'objet serveur.
* :bug:`153` Les champs optionnels peuvent désormais être mis à jour avec une
  valeur vide. Précédemment, une valeur avait été donné à un champ optionnel
  (par exemple les divers chemins des règles) au moment de la création, il était
  impossible de supprimer cette valeur par la suite (à moins de supprimer l'objet
  puis de le réinsérer).
* :feature:`153` Dans l'API REST, les méthodes ``PUT`` et ``PATCH`` ont désormais
  des *handlers* distincts, avec des comportements différents. La méthode ``PATCH``
  permet de faire une mise-à-jour partielle de l'objet ciblé (les champs omits
  resteront inchangés). La méthode ``PUT`` permet, elle, de remplacer intégralement
  toutes les valeurs de l'objet (les champs omits n'auront donc plus de valeur
  si le modèle le permet).
* :bug:`179` Corrige la commande de lancement des transferts avec Waarp R66
* :bug:`188` Correction de l'erreur 'bad file descriptor' du CLI lors de
  l'affichage du prompt de mot de passe sous Windows
* :feature:`169` En cas d'absence du nom d'utilisateur, celui-ci sera demandé
  via un prompt du terminal
* :feature:`169` Le paramètre de l'adresse de la gateway dans les commandes du
  client terminal peut désormais être récupérée via la variable d'environnement
  ``WAARP_GATEWAY_ADDRESS``. En conséquence de ce changement, le paramètre a été
  changé en option (``-a``) et est maintenant optionnel. Pour éviter les
  confusions entre ce nouveau flag et l'option ``--account`` déjà existante sur
  la commande `transfer add`, cette dernière a été changée en ``-l`` (ou
  ``--login`` en version longue).

* :release:`0.2.0 <2020-08-24>`
* :feature:`178` Redémarre le automatiquement le service si celui-ci était
  démarré après l'installation d'une mise à jour via les packages DEB/RPM
* :bug:`171` Correction d'une erreur de pointeur nul lors de l'arrêt d'un serveur SFTP déjà arrêté
* :bug:`159` Sous Unix, par défaut, le programme cherche désormais le fichier de configuration ``gatewayd.ini`` dans le dossier ``/etc/waarp-gateway/`` au lieu de ``/etc/waarp/``
* :feature:`158` Sous Windows, le programme cherchera le fichier de configuration ``gatewayd.ini`` dans le dossier ``%ProgramData%\waarp-gateway`` si aucun chemin n'est renseigné dans la commande le lancement (en plus des autres chemins par défaut)
* :bug:`161` Correction de la forme longue de l'option ``--password`` de la commande ``remote account update``
* :feature:`157` L'option ``-c`` est désormais optionnelle pour les commandes d'import/export (similaire à la commande ``server``)
* :bug:`162` L'API REST et le CLI renvoient désormais la liste correcte des partenaires/serveurs/comptes autorisés à utiliser une règle
* :bug:`165` Correction des incohérences de capitalisation dans le sens des règles
* :bug:`160` Correction de l'erreur 'record not found' lors de l'appel de la commande ``history retry``
* :bug:`156` Correction des paramètres d'ajout et d'update des rules pour tenir compte des in, out et work path
* :bug:`155` Correction de l'erreur d'update partiel des local/remote agents lorsque protocol n'est pas fourni
* :bug:`154` Correction de l'erreur de l'affichage du workpath des règles
* :bug:`152` Correction de l'erreur de timeout du CLI lorsque l'utilisateur met plus de 5 secondes à entrer le mot de passe via le prompt

* :release:`0.1.0 <2020-08-19>`
* :feature:`-` Première version publiée

