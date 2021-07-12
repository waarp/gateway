.. _changelog:

Historique des versions
=======================

* :bug:`133` Correction d'une erreur rendant impossible la répartition de charge
  sur plusieurs instances d'une même *gateway*. Précédemment, il était possible
  pour 2 instances d'une même *gateway* de récupérer un même transfert depuis la
  base de données, et de l'exécuter 2 fois en parallèle. Ce n'est désormais plus
  possible.
* :feature:`` Un champ `passwordHash` a été ajouté à l'objet JSON de compte local
  du fichier d'import/export. Il remplace le champ `password` pour l'export de
  configuration. La gateway ne stockant que des hash de mots de passe, le nom du
  champ n'était pas approprié. Le champ `password` reste cependant utilisable
  pour l'import de fichiers de configuration généré par des outils tiers.
* :bug:`` Les champs optionnels vides ne seront désormais plus ajouté aux fichiers
  de sauvegarde lors d'un export de configuration.
* :bug:`252` Les certificats, clés publiques & clés privées sont désormais parsés
  avant d'être insérés en base de données. Les données invalides seront désormais
  refusées.
* :bug:`` Correction d'une régression empêchant le redémarrage des transferts SFTP.
* :feature:`242` Ajout de la direction (`isSend`) à l'objet *transfer* de REST.
* :bug:`239` Correction d'une erreur de base de données survenant lors de la mise
  à jour de la progression des transferts.
* :bug:`222` Correction d'un comportement incorrect au lancement de la *gateway*
  lorsque la racine `GatewayHome` renseignée est un chemin relatif.
* :bug:`238` Suppression de l'option (maintenant inutile) ``R66Home`` du fichier
  de configuration.
* :bug:`254` Ajout des contraintes d'unicité manquantes lors de l'initialisation
  de la base de données.
* :bug:`` Les dates de début/fin de transfert sont désormais précises à la
  milliseconde près (au lieu de la seconde).
* :bug:`243` Correction d'un bug empêchant l'annulation d'un transfert avant
  qu'il n'ait commencé car sa date de fin se retrouvait antérieure à sa date de
  début. Par conséquent, désormais, en cas d'annulation, la date de fin du
  transfert sera donc nulle.

* :release:`0.3.3 <2021-04-07>`
* :bug:`251` Corrige le probème de création du fichier distant en SFTP
  lorsque le serveur refuse l'ouverture de fichier en écriture ET en lecture.
* :bug:`251` Corrige un problème du script d'update-conf qui sort en erreur
  si les fichiers optionnels ne sont pas dans l'archive de déploiement.

* :release:`0.3.2 <2021-04-06>`
* :bug:`248` Ajout de l'option `insecure` au client terminal afin de désactiver la
  vérification des certificats serveur https.

* :release:`0.3.1 <2021-01-25>`
* :bug:`241` Correction du typage de la colonne `permissions` de la table `users`.
  La colonne est désormais de type *BINARY* (au lieu de *INT*).

* :release:`0.3.0 <2020-12-14>`
* :bug:`213` Correction d'une erreur causant la suppression des post traitements
  et des traitements d'erreur lors de la mise à jour d'une règle.
* :bug:`211` Correction d'une erreur causant le changement de la direction d'une
  règle lors d'un *update* via l'interface REST.
* :bug:`212` Correction du comportement des méthodes SFTP ``List`` et ``Stat``.
  Les substitutions de chemin se font désormais correctement, même lorsque la
  règle n'a pas de ``in/out_path``. Les fichiers pouvant être téléchargés depuis
  le serveur SFTP sont donc maintenant visibles via ces 2 méthodes. Les fichiers
  entrants, en revanche, ne seront pas visibles une fois déposés.
* :feature:`219` Le chemin (``path``) n'est plus obligatoire lors de la création
  d'une règle. Par défaut, le nom de la règle sera utilisé comme chemin (les
  règles d'unicité sur le chemin s'applique toujours).
* :bug:`219` Il est désormais possible de créer 2 règles avec des chemins
  (``path``) identiques si leur directions sont différentes.
* :bug:`221` Ajout de l'identifiant de transfert distant aux interfaces REST &
  terminal. Lorsqu'un agent de transfert se connecte à la *gateway* pour faire
  un transfert, cet identifiant correspond au numéro que cet agent a donné au
  transfert, et qui est donc différent de l'identifiant que la *gateway* a donné
  à ce transfert.
* :bug:`216` Ajout de l'adresse manquante lors de l'export d'agents locaux/distants.
* :bug:`218` Correction d'une erreur où le client de transfert envoyait le premier
  packet de données en boucle lorsque la taille du fichier dépassait la taille
  d'un packet.
* :bug:`217` Correction d'une erreur causant un *panic* du serveur dans certaines
  circonstances à la fin d'un transfert.
* :bug:`215` Correction d'une erreur de typage des identifiants de transfert R66.
* :bug:`176` Les arguments de direction de transfert du client terminal ont été
  rendu consistants entre les différentes commandes. Le sens d'un transfert
  s'exprime désormais toujours avec les mots ``send`` et ``receive`` (en minuscules)
  pour toutes les commandes.
* :feature:`131` Ajout d'un système de gestion des droits pour les utilisateurs
  de l'interface d'administration. Les utilisateurs de la *gateway* ont désormais
  des droits attachés permettant de restreindre les actions qu'ils sont autorisés
  à effectuer via l'interface REST. Cette gestion des droits peut se faire via
  la commande de gestion des utilisateurs du client terminal, ou via l'interface
  REST de gestion des utilisateurs directement.
* :bug:`210` Les mots de passe des serveurs R66 locaux renseignés dans la
  configuration protocolaire sont désormais cryptés avant d'être stockés en base,
  au lieu d'être stockés en clair. Le stockage (sous forme de hash) des mots de
  passe des serveurs R66 distants reste inchangé.
* :feature:`208` L'option du CLI pour entrer la configuration protocolaire d'un
  serveur ou d'un partenaire (``-c``) a été changée. La configuration doit
  désormais être entrée sous la forme ``-c clé:valeur``, répétée autant de fois
  qu'il y a de valeurs dans la configuration.
* :bug:`208` Le mot de passe des serveurs R66 renseigné dans la configuration
  protocolaire ne doit plus être encodé en base64 pour être accepté par l'API REST.
* :bug:`208` Les mots de passe des utilisateurs & des comptes locaux/distants
  ne doivent plus être encodés en base64 pour être acceptés par l'API REST.
* :bug:`207` Correction d'une erreur où les mots de passe des partenaires R66
  distants n'étaient pas correctement hashés.
* :bug:`205` Correction d'une erreur empêchant le démarrage des serveurs R66 locaux.
* :bug:`206` Correction d'une erreur causant un double hachage du mot de passe
  du client R66.
* :bug:`201` Correction du typage de la colonne `step` des tables `transfers` et
  `transfer_history`. La colonne est désormais de type *VARCHAR* (au lieu de *INT*).
* :bug:`200` Les écritures de la progression du transfert de données se fait
  désormais à intervalles réguliers (1 fois par seconde) au lieu de que ce soit
  à chaque écriture sur disque. Cela devrait grandement réduire le nombre
  d'écritures en base de données lors d'un transfert, notamment pour les gros fichiers.
* :bug:`` Correction d'un bug dans le serveur SFTP qui causait le déplacement
  du fichier temporaire de réception vers son chemin final malgré le fait qu'une
  erreur ait survenue durant le transfert de données.
* :bug:`` Lors d'un transfert SFTP entrant, le fichier (temporaire) de destination
  est désormais créé lors de la réception du 1er packet de données, au lieu du
  packet de requête.
* :bug:`199` Correction d'un bug qui causait une double fermeture des fichiers
  de transfert, ce qui causait l'apparition d'une *warning* dans les logs sur
  lequel l'utilisateur ne pouvait pas agir.
* :feature:`129` Ajout d'un client et d'un serveur R66 à la *gateway*. Il est
  donc désormais possible d'effectuer des transferts R66 sans avoir recours à un
  serveur externe.
* :bug:`` Lors d'un transfert, le compteur ``task_number`` est désormais
  réinitialisé lors du passage à l'étape suivante au lieu de la fin de la chaîne
  de traitements.
* :feature:`` Afin de faciliter la reprise de transfert, les transferts en erreur
  resteront désormais dans la table ``transfers`` au lieu d'être déplacés dans
  la table ``transfer_history``. Cette dernière ne contiendra donc que les
  transferts terminés ou annulés. Ce changement a 2 conséquences:
  - Il est désormais possible de redémarrer n'importe quel transfert de l'historique
    via la commande ``history retry`` (ou le point d'accès REST ``/api/history/{id}/retry``).
    En revanche, ceux-ci reprendront dorénavant depuis le début avec un nouvel
    identifiant.
  - La reprise des transferts en erreur se fait désormais via la commande
    ``transfer resume`` (ou le point d'accès REST ``/api/transfer/{id}/resume``).
* :feature:`` La colonne ``ext_info`` a été supprimée des tables ``transfers`` &
  ``transfer_history``, et une nouvelle table ``transfer_info`` a été créée à la
  place. Cette table permet d'associer un ensemble de clés & valeurs arbitraires
  à un transfert.
* :bug:`` Retrait de l'auto-incrément sur la colonne ``id`` de la table
  ``transfer_history`` qui causait l'attribution d'un identifiant erroné au
  transfert lors de son insertion dans la table d'historique.
* :bug:`197` Un transfert dont le temps d'exécution est supérieur à la durée
  d'attente du controller pouvait être exécuté plusieurs fois
* :feature:`173` L'adresse (et le port) des serveurs & partenaires a été extrait
  de la colonne de configuration protocolaire, et 1 nouvelle colonne ``address``
  contenant l'adresse de l'agent a été ajoutée au tables ``local_agents`` &
  ``remote_agents``.
* :bug:`173` La présence de champs inconnus dans la configuration protocolaire
  des partenaires & serveurs produit désormais une erreur (au lieu d'être ignorée).
* :feature:`173` Dans l'API REST, les objets JSON partenaire & serveur ont
  désormais un champ ``address`` contenant l'adresse de l'agent.
* :feature:`173` Dans le CLI, les sous-commandes ``add`` & ``update`` des
  commandes ``server`` & ``partner`` possèdent désormais un paramètre ``-a``
  indiquant l'adresse du serveur/partenaire. Les sous-commandes ``add`` & ``list``
  affichent également l'adresse du serveur/partenaire désormais.
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
* :bug:`193` Les transferts SFTP peuvent désormais être redémarrés via la commande
  ``retry``. (Attention: lorsque la gateway agit en tant que serveur, redémarrer
  un transfert créera une nouvelle entrée au lieu de reprendre l'ancienne, il est
  donc déconseillé de redémarrer le transfert dans ce cas.)
* :bug:`180` Ajout de commande versions au serveur et au client
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

