.. _changelog:

Historique des versions
=======================

* :bug:`291` Correction d'une erreur causant l'apparition impromptue de messages
  d'erreur (*warnings*) lorsqu'un client SFTP termine normalement une connexion
  vers un serveur SFTP de la *gateway*.
* :feature:`345` Les erreurs pouvant survenir lors de l'interruption ou
  l'annulation d'un transfert sont dorénavant correctement loggées. Par ailleurs,
  il est désormais possible d'annuler un transfert en cours, et ce, même si la
  *pipeline* responsable de son exécution ne peut être trouvée. En cas de problème,
  cela devrait permettre d'éviter que des transferts restent bloqués indéfiniment.
* :feature:`225` Ajout d'une option 'TLSPassphrase' à la section 'Admin' du
  fichier de configuration. Cela permet de renseigner le mot de passe de la
  clé privé (passphrase) du serveur d'administration si celle-ci est chiffrée.
  Il est donc désormais possible d'utiliser une clé privée chiffrée pour le
  certificat TLS du serveur d'administration.
* :feature:`285` Ajout d'une option ``-r, --reset-before-import`` à la commande
  d'import. Quand présente, cette option indique à la *gateway* que la base de
  données doit être vidée avant d'effectuer l'import. Ainsi, tous les éléments
  présents en base concernés par l'opération d'import seront supprimés. Une 2nde
  option nommée ``--force-reset-before-import`` a été ajoutée, permettant aux
  scripts d'outrepasser le message de confirmation de l'option ``-r``.
* :feature:`224` Ajout des utilisateurs *gateway* au fichier d'import/export.
  Il est désormais possible d'exporter et importer les utilisateurs *gateway*
  servant à l'administration. Par conséquent, l'option ``-t --target`` des
  commandes :ref:`reference-cmd-waarp-gatewayd-import` et
  :ref:`reference-cmd-waarp-gatewayd-export` accepte
  désormais la valeur ``users``.

* :release:`0.6.2 <2022-08-22>`
* :bug:`343` Il était impossible de migrer la base de données vers la version
  0.6.1.

* :release:`0.6.1 <2022-08-18>`
* :bug:`340` Correction d'une erreur causant l'échec des migrations de base de
  données due à une mauvaise prise en compte du fichier de configuration.
* :bug:`341` La commande de listing des partenaires liste correctement les
  partenaires au lieu des serveurs locaux.

* :release:`0.6.0 <2022-07-22>`
* :bug:`337` La tâche *TRANSFER* n'utilise plus la même arborescence en local et
  en distant lors de la programmation d'un transferts. Cela pouvait causer des
  problèmes lorsque les deux arborescences n'étaient pas similaires.
* :bug:`338` Le sens de transfert renvoyé par l'API REST est désormais correct
  (précédemment, tous les transferts étaient marqués comme étant en réception).
* :bug:`-` Correction d'une erreur *'account not found'* pouvant survenir lors
  d'un import de configuration si la base de données est partagée entre plusieurs
  agents.
* :bug:`-` Correction d'un *panic* qui pouvait survenir lorsqu'une commande du
  CLI était exécutée avec l'option `-i, --insecure`.
* :feature:`256` Ajout du listing de fichiers et de la requête de métadonnées de
  transferts au serveur R66 de la gateway. Il est désormais possible pour un
  client R66 de demander au serveur une liste des fichiers transférables avec
  une règle données. Il est également possible désormais pour un client de
  demander des informations sur un transfert qu'il a effectué avec le serveur.
* :feature:`250` Ajout du support des *transfers info* à la gateway. Les
  *transfer info* sont une liste de paires clé-valeur définies par l'utilisateur
  à la création du transfert, et qui seront envoyées par le client en même temps
  que la requête, pour les protocoles le permettant, à savoir R66 et HTTP pour
  l'instant.

* :release:`0.5.2 <2022-06-30>`
* :bug:`319` Lorsqu'un protocole n'intègre pas de mécanisme pour négocier une
  reprise de transfert, alors le transfert de données est repris depuis le début.
  Cela permet d'éviter que dans certains cas, le fichier envoyé soit incomplet
  après une reprise de transfert.
* :bug:`` Correction d'un bug pouvant causer un deadlock lorsqu'une erreur se
  produit durant un transfert R66.
* :bug:`315` Lorsqu'un transfert est interrompu durant l'envoi de données, et que
  le transfert est redémarré, l'envoi de données reprendra depuis le début du
  fichier, à moins que le protocole de transfert intègre un mécanisme permettant
  une négociation sur l'endroit d'où reprendre le transfert (comme c'est le cas
  pour R66 par exemple). Cela permet d'éviter qu'un fichier soit potentiellement
  envoyé avec des parties manquantes.
* :bug:`329` Correction de l'impossibilité pour la *gateway* de se connecter via
  R66-TLS à un agent *Waarp-R66*. Une exception a été ajoutée pour le certificat
  de *Waarp-R66* afin que celui-ci soit accepté par la *gateway* (voir
  :ref:`les détails d'implémentation R66<ref-proto-r66>` pour plus d'informations).
* :bug:`326` Les fichiers transférés ne sont plus requis de se trouver immédiatement
  dans le dossier de la règle avec laquelle ils sont transférés. Il est désormais
  possible de transférer des fichiers se trouvant dans des sous-dossiers.
* :bug:`318` Dépréciation de tous les algorithmes de signature TLS basés sur SHA1.
  Les certificats signés avec SHA1 sont encore acceptés pour le moment mais seront
  systématiquement refusés dans les versions futures.
* :bug:`330` Correction de l'option ``-c --config`` de la commande ``partner add``
  pour qu'elle ait le même comportement que sur les autres commandes similaires.
  L'option peut maintenant être répétée pour chaque paramètre supplémentaire,
  comme mentionné dans :doc:`la documentation<reference/cli/client/partner/add>`
  de la commande.
* :bug:`315` Les erreurs survenant lors de l'initialisation du transfert sont
  maintenant correctement gérées. Précédemment, la mauvaise gestion de ces
  erreur pouvait conduire un transfert à se retrouver dans le mauvais statut
  lorsqu'une erreur se produisait.
* :bug:`328` Correction d'une erreur pouvant causer des collisions d'identifiants
  de transfert lorsque l'incrément de la base de données est réinitialisé. La
  *gateway* génère dorénavant un identifiant de transfert unique (le
  *RemoteTransferID*) qui est envoyé dans la requête de transfert à la place de
  l'ancien auto-incrément. L'identifiant auto-incrémenté reste disponible à des
  fins d'administration.

* :release:`0.5.1 <2022-04-26>`
* :bug:`322` Correction d'une erreur `provided data is not a pointer to struct`
  survenant lors de l'appel au client *waarp-gateway*.

* :release:`0.5.0 <2022-04-14>`
* :bug:`309` Génération et publication d'images Docker
* :bug:`311` Correction d'une erreur du client SFTP pouvant survenir lorsque
  celui-ci effectue un transfert vers un serveur configuré en lecture unique
  (*read-once*). Pour cela, 2 nouvelles options ``useStat`` et
  ``disableClientConcurrentReads`` ont été ajoutée à la
  :ref:`configuration protocolaire SFTP<proto-config-sftp>`
* :bug:`304` Correction d'un bug de blocage de transfert dû à un problème
  de concurrence pouvant survenir lors de l'interruption d'un transfert.
* :feature:`306` Ajout de l'attribut ``protocol`` à l'objet JSON de transfert.
  Cela permet plus de consistance avec l'objet d'historique qui contenait déjà
  cet attribut. Le protocole est également visible désormais en sortie de la
  commande ``transfer get`` du terminal.
* :bug:`-` Correction d'une erreur SIGSEGV survenant lors de l'exécution d'une
  commande su client terminal sans que l'adresse de la *gateway* soit renseignée.
  Désormais, le client lèvera une erreur plus claire au lieu de paniquer.
* :bug:`307` Correction d'une erreur *"context canceled"* pouvant survenir lors
  de l'exécution de certaines commandes du client terminal.
* :bug:`302` Correction d'une erreur du serveur R66 causée par le fait que le
  serveur ne prenait pas en compte certaine partie de sa *ProtoConfig*. Cela causait
  par exemple le démarrage du serveur en clair lorsqu'aucun certificats n'était
  trouvé, et ce, malgré le fait que le serveur soit configuré pour opérer avec TLS.
* :bug:`301` Correction d'une erreur de création des dossiers in/out/temp au lancement
  de la gateway.
* :feature:`300` Correction d'une erreur du client terminal dans la commande de
  création et de mise à jour des règles de transfert. Si le JSON définissant une
  tâche était invalide, celui-ci était ignoré au lieu qu'une erreur soit levée,
  et la règle était simplement ajoutée sans cette tâche. Désormais, un JSON de
  tâche invalide produira une erreur comme attendu.
* :feature:`268` Ajout d'un fichier *override* permettant à une instance de
  *gateway* au sein d'une grappe d'écraser localement certaines parties de la
  configuration globale de la grappe (voir :ref:`la documentation<reference-conf-override>`
  du fichier d'override de configuration pour plus de détails).
  Pour l'heure, ce fichier permet de définir des remplacement d'adresses pour les
  serveurs locaux, ce qui est nécessaire pour que la *gateway* fonctionne
  correctement en grappe.
* :bug:`275` Correction d'une erreur empêchant l'acceptation de transfert de
  fichier vide via R66.
* :feature:`274` Les contraintes d'unicité déclarées dans les scripts de migration
  de la base de données sont désormais via des indexes uniques, au lieu des
  contraintes sur les colonnes. Le module de migration est désormais consistant
  avec le module d'initialisation de la base sur ce point.
* :bug:`292` Correction d'une erreur empêchant la création de l'utilisateur par
  défaut lorsque la base de données est partagée entre plusieurs *gateways*.
* :bug:`-` Correction d'un bug permettant la suppression du dernier administrateur
  d'une *gateway*, rendant cette dernière impossible à administrer.
* :bug:`294` Correction d'une erreur dans la réponse des requêtes de listage
  d'utilisateurs sur l'interface REST d'administration (et le client terminal).
  Lorsque la base de données est partagée entre plusieurs *gateways*, l'interface
  d'administration renvoyait indistinctement les utilisateur de toutes les
  *gateways* utilisant cette base de données, au lieu de renvoyer uniquement les
  utilisateurs de l'instance interrogée. Désormais, l'interface REST ne renvoi que
  les utilisateurs de la *gateway* interrogée. Un problème similaire a également
  été corrigé pour les transferts.
* :feature:`277` Ajout d'une option à la commande `history list` de la CLI
  permettant de trier les entrées de l'historique par date de fin (`stop+` et
  `stop-`). Cette option est également présente sur l'API REST de la *gateway*.
* :bug:`278` Dans le fichier d'import, si une des listes définissant les chaînes
  de traitements de la règle (``pre``, ``post`` ou ``error``) est vide mais non-nulle,
  la chaîne de traitements en question sera vidée. Si la liste est manquante ou
  nulle, la chaîne de traitements restera inchangée.
* :feature:`270` Lors d'une requête SFTP, la recherche de la règle associée au
  chemin de la requête se fait désormais récursivement, au lieu de juste prendre
  le dossier parent. Cela a les conséquences suivantes:

  - il est désormais possible d'ajouter des sous-dossiers à l'intérieur du dossier
    d'une règle
  - la commande SFTP `stat` fonctionne désormais correctement sur les dossiers
  Pour que cela soit possible, les changements suivants ont été nécessaires:

  - les chemins de règles ne sont plus stockés avec un '/' au début
  - le chemin d'une règle ne peut plus être parent du chemin d'une autre règle
    (par exemple, une règle `/toto/tata` ne peut exister en même temps qu'une
    règle `/toto` car cela créerait des conflits)
* :bug:`-` Les chemins de règle (*path*) ne sont désormais plus stockés avec le
  '/' de début.
* :feature:`247` Ajout d'un client et d'un serveur HTTP/S à la *gateway*. Il est
  donc désormais possible d'effectuer des transferts via ces 2 protocoles.
* :feature:`194` Dépréciation des champs REST ``sourceFilename`` et ``destFilename``
  de l'objet JSON *history*, remplacés par les champs ``localFilepath`` et
  ``remoteFilepath``.
* :feature:`194` Dépréciation des champs REST ``inPath`` et ``outPath`` de l'objet
  JSON *rule*, remplacés par les champs ``localDir`` et ``remoteDir``. Le champ
  ``workPath`` du même objet est également déprécié, remplacé par le champ
  ``tmpLocalRcvDir``. Ces champs ont également été dépréciés dans le fichier JSON
  d'import/export. Les nouveaux champs de remplacement sont identiques à ceux de
  REST.

  Les options de commande correspondantes du CLI ont également été dépréciées.
  Ainsi, les options ``-i, --in_path`` et ``-o, --out_path`` des commandes
  ``rule add`` et ``rule update`` ont été remplacées par les options
  ``--local-dir`` et ``--remote-dir``. L'option ``-w, --work_path`` a, elle, été
  remplacée par ``--tmp-dir``.

* :feature:`194` Dépréciation des champs REST ``root``, ``inDir``, ``outDir`` et
  ``workDir`` de l'objet JSON *server*, remplacés respectivement par ``rootDir``,
  ``receiveDir``, ``sendDir`` et ``tmpReceiveDir``. Ces champs ont également été
  dépréciés dans le fichier JSON d'import/export. Les nouveaux champs de
  remplacement sont identiques à ceux de REST.

  Les options de commande correspondantes du CLI ont également été dépréciées.
  Ainsi, les options ``-r, --root``, ``-i, --in``, ``-o, --out`` et ``-w, --work``
  des commandes ``server add`` et ``server update`` ont été remplacées respectivement
  par les options ``--root-dir``, ``--receive-dir``, ``--send-dir`` et ``--tmp-dir``.
* :feature:`194` Dépréciation des champs REST ``trueFilepath``, ``sourcePath``
  et ``destPath`` de l'objet JSON *transfer*, remplacés par les champs
  ``localFilepath`` et ``remoteFilepath``. Le champ ``startDate`` du même objet
  est également déprécié en faveur du champ ``start``.

  De plus, l'option ``-n, --name`` de la commande ``transfer add`` est dépréciée
  en faveur de l'option ``-f, --file`` déjà existante.

* :release:`0.4.4 <2021-10-25>`
* :bug:`282` Correction d'un bug dans le moteur de migration de base de données
  qui laissait la base dans un état inutilisable après une migration à cause
  d'une disparité de version entre la base et l'exécutable.

* :release:`0.4.3 <2021-09-24>`
* :bug:`-` Activation des migrations de base de données vers la version 0.4.2
* :bug:`-` Correction de la compilation avec certaines versions de Go

* :release:`0.4.2 <2021-09-21>`
* :bug:`273` Correction d'une erreur "database table locked" pouvant survenir
  lorsqu'une base de données SQLite est partagée entre plusieurs instances de
  *gateway*.
* :bug:`272` Correction d'une erreur pouvant survenir lors de l'import d'un
  serveur local dont le nom existe déjà sur une autre instance de *gateway*
  partageant la même base de données.
* :bug:`263` Suppression du '/' présent au début des noms de dossiers renvoyés
  lors de l'envoi d'une commande SFTP *ls* . Cela devrait résoudre un certains
  nombre de problèmes survenant lors de l'utilisation de cette commande.
* :bug:`265` Correction d'un bug causé par une contrainte d'unicité sur la table
  d'historique.
* :bug:`266` Correction d'une erreur dans les authorisations de règles renvoyées
  via l'API REST. Les authorisations renvoyées devraient désormais être correctes.
* :bug:`267` Correction d'une erreur permettant de démarrer un serveur SFTP même
  quand celui-ci n'a pas de *hostkey*, empêchant ainsi toute connexion à ce
  serveur. Dorénavant, l'utilisateur sera informé de cette absence de *hostkey*
  au démarrage du serveur (et non lors de la connexion à celui-ci).

* :release:`0.4.1 <2021-07-21>`
* :bug:`-` La *gateway* refusera désormais de démarrer si la version de la base
  de données est différente de celle du programme.

* :release:`0.4.0 <2021-07-21>`
* :bug:`259` Correction d'un bug causant une erreur après les pré-tâches d'un
  transfer R66 côté serveur.
* :bug:`260` Correction d'une erreur dans l'import des mots de passe de comptes
  locaux R66.
* :bug:`133` Correction d'une erreur rendant impossible la répartition de charge
  sur plusieurs instances d'une même *gateway*. Précédemment, il était possible
  pour 2 instances d'une même *gateway* de récupérer un même transfert depuis la
  base de données, et de l'exécuter 2 fois en parallèle. Ce n'est désormais plus
  possible.
* :bug:`-` Sous système Unix, l'interruption de tâches externes se fait désormais
  via un *SIGINT* (au lieu de *SIGKILL*).
* :feature:`-` Ajout d'un champ taille de fichier ``filesize`` au modèles de
  transfert et d'historique.
* :feature:`-` Il n'est plus obligatoire pour un partenaire SFTP d'avoir une
  *hostkey* (certificat) pour pouvoir créer un transfert vers/depuis cet agent.
  Une *hostkey*, reste nécessaire pour les transferts SFTP, mais la vérification
  sera désormais faite au démarrage du transfert (au lieu de son enregistrement).
* :feature:`-` Dépréciation des options ``InDirectory``, ``OutDirectory`` &
  ``WorkDirectory`` du fichier de configuration de la *Gateway*. Ces options ont
  été remplacés respectivement par ``DefaultInDir``, ``DefaultOutDir`` &
  ``DefaultTmpDir``.
* :feature:`-` Dépréciation des champs JSON ``inDir``, ``outDir`` & ``workDir`` de
  l'objet REST de serveur local. Les champs ont été remplacé par ``serverLocalInDir``,
  ``serverLocalOutDir`` & ``serverLocalTmpDir`` représentant respectivement le
  dossier de réception du serveur, le dossier d'envoi du serveur, et le dossier
  de réception temporaire.
* :feature:`-` Dépréciation des champs JSON ``inPath``, ``outPath`` & ``workPath``
  de l'objet REST de règle. Les champs ont été remplacé par ``localDir``,
  ``remoteDir`` & ``localTmpDir`` représentant respectivement le dossier sur le
  disque local de la *Gateway*, le dossier sur l'hôte distant, et le dossier
  temporaire local.
* :feature:`-` Dépréciation des champs JSON ``sourcePath``, ``destPath`` & ``trueFilepath``
  des objets REST de consultation des transferts et de l'historique. Ces champs ont été
  remplacé par les champs ``localPath`` & ``remotePath`` contenant respectivement
  le chemin du fichier sur le disque local de la *Gateway*, et le chemin d'accès au
  fichier sur l'hôte distant.
* :feature:`-` Dépréciation des champs ``sourcePath`` & ``destPath`` des objets
  REST de création de transfert. Ces champs ont été remplacé par le champ
  ``file`` contenant le nom du fichier à transférer. Il ne sera donc, à terme,
  plus possible de donner au fichier de destination du transfer un nom différent
  de celui du fichier source.
* :feature:`-` Un champ `passwordHash` a été ajouté à l'objet JSON de compte local
  du fichier d'import/export. Il remplace le champ `password` pour l'export de
  configuration. La gateway ne stockant que des hash de mots de passe, le nom du
  champ n'était pas approprié. Le champ `password` reste cependant utilisable
  pour l'import de fichiers de configuration généré par des outils tiers.
* :bug:`-` Les champs optionnels vides ne seront désormais plus ajouté aux fichiers
  de sauvegarde lors d'un export de configuration.
* :bug:`252` Les certificats, clés publiques & clés privées sont désormais parsés
  avant d'être insérés en base de données. Les données invalides seront désormais
  refusées.
* :bug:`-` Correction d'une régression empêchant le redémarrage des transferts SFTP.
* :feature:`242` Ajout de la direction (`isSend`) à l'objet *transfer* de REST.
* :bug:`239` Correction d'une erreur de base de données survenant lors de la mise
  à jour de la progression des transferts.
* :bug:`222` Correction d'un comportement incorrect au lancement de la *gateway*
  lorsque la racine `GatewayHome` renseignée est un chemin relatif.
* :bug:`238` Suppression de l'option (maintenant inutile) ``R66Home`` du fichier
  de configuration.
* :bug:`254` Ajout des contraintes d'unicité manquantes lors de l'initialisation
  de la base de données.
* :bug:`-` Les dates de début/fin de transfert sont désormais précises à la
  milliseconde près (au lieu de la seconde).
* :bug:`243` Correction d'un bug empêchant l'annulation d'un transfert avant
  qu'il n'ait commencé car sa date de fin se retrouvait antérieure à sa date de
  début. Par conséquent, désormais, en cas d'annulation, la date de fin du
  transfert sera donc nulle.
* :feature:`242` Ajout de la direction (`isSend`) à l'objet *transfer* de REST.

* :release:`0.3.3 <2021-04-07>`
* :bug:`251` Corrige le problème de création du fichier distant en SFTP
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
* :bug:`-` Correction d'un bug dans le serveur SFTP qui causait le déplacement
  du fichier temporaire de réception vers son chemin final malgré le fait qu'une
  erreur ait survenue durant le transfert de données.
* :bug:`-` Lors d'un transfert SFTP entrant, le fichier (temporaire) de destination
  est désormais créé lors de la réception du 1er packet de données, au lieu du
  packet de requête.
* :bug:`199` Correction d'un bug qui causait une double fermeture des fichiers
  de transfert, ce qui causait l'apparition d'une *warning* dans les logs sur
  lequel l'utilisateur ne pouvait pas agir.
* :feature:`129` Ajout d'un client et d'un serveur R66 à la *gateway*. Il est
  donc désormais possible d'effectuer des transferts R66 sans avoir recours à un
  serveur externe.
* :bug:`-` Lors d'un transfert, le compteur ``task_number`` est désormais
  réinitialisé lors du passage à l'étape suivante au lieu de la fin de la chaîne
  de traitements.
* :feature:`-` Afin de faciliter la reprise de transfert, les transferts en erreur
  resteront désormais dans la table ``transfers`` au lieu d'être déplacés dans
  la table ``transfer_history``. Cette dernière ne contiendra donc que les
  transferts terminés ou annulés. Ce changement a 2 conséquences:

  - Il est désormais possible de redémarrer n'importe quel transfert de l'historique
    via la commande ``history retry`` (ou le point d'accès REST ``/api/history/{id}/retry``).
    En revanche, ceux-ci reprendront dorénavant depuis le début avec un nouvel
    identifiant.
  - La reprise des transferts en erreur se fait désormais via la commande
    ``transfer resume`` (ou le point d'accès REST ``/api/transfer/{id}/resume``).
* :feature:`-` La colonne ``ext_info`` a été supprimée des tables ``transfers`` &
  ``transfer_history``, et une nouvelle table ``transfer_info`` a été créée à la
  place. Cette table permet d'associer un ensemble de clés & valeurs arbitraires
  à un transfert.
* :bug:`-` Retrait de l'auto-incrément sur la colonne ``id`` de la table
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

