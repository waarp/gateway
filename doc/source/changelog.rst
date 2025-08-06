.. _changelog:

Historique des versions
=======================

* :feature:`-` Correction du mode de compatibilité non-standard de PeSIT.
  Celui-ci gère maintenant correctement les noms de fichiers en émission et en
  réception. Les modes de compatibilité pour PeSIT ont par ailleurs été renommés
  en "standard" et "non-standard", ce dernier servant pour la compatibilité avec
  les applications PeSIT tierces.
* :feature:`-` Le découpage en article des transferts PeSIT est désormais
  correctement traité et stocké dans les infos de transfert.
* :feature:`464` Il est désormais possible de préenregistrer un transfert serveur.
  Préenregistrer un transfert permet, entre autres, de conserver les informations
  de transfert dans le cas d'un rebond vers un transfert serveur. Cela permet
  également de spécifier une date limite de disponibilité pour un fichier.
  Un transfert serveur peut être préenregistré via la commande terminal
  :ref:`"transfer preregister"<ref-cli-transfer-preregister>` ou bien via la
  nouvelle tâche :ref:`PREGEGISTER <ref-tasks-preregister>`.
* :feature:`467` Certains attributs PeSIT sont désormais stockés sous forme
  d'informations de transfert. Ces informations sont : l'encodage du fichier,
  le type de fichier, l'organisation du fichier, l'identifiant de banque et
  l'identifiant de client. Pour plus de détails, voir la page sur
  :ref:`l'implémentation de PeSIT<ref-proto-pesit>` dans la Gateway.
* :bug:`-` Correction d'une erreur du serveur REST faisant que les entêtes
  "Server" et "Waarp-Gateway-Date" n'étaient pas correctement renvoyés en
  réponse aux requêtes faites sur l'API REST.
* :feature:`-` Ajout d'une option ``-r, --raw`` aux commandes d'affichage des
  certificats et clés SSH permettant d'afficher la valeur brute de ces éléments
  (typiquement un fichier PEM) au lieu d'afficher leurs métadonnées comme c'est
  le cas par défaut.
* :feature:`448` Ajout d'une tâche "EMAIL" permettant d'envoyer un email.
  Particulièrement utile en tant que tâche d'erreur pour notifier d'une erreur
  de transfert. Pour configurer cette tâche, deux nouvelles tables, ainsi que
  leurs :ref:`handlers REST<ref-rest-emails>` et leurs :ref:`commandes terminal
  <ref-cli-client-email>` ont été ajoutées. Ces tables permettent de configurer
  les identifiants de connexion SMTP, ainsi que les templates d'email à envoyer.
* :feature:`435` Ajout d'une commande CLI et d'un handler REST permettant
  d'envoyer des notifications SNMP de test afin de valider la configuration des
  moniteurs SNMP.
* :feature:`438` Ajout du nom de l'instance Gateway dans les *traps* SNMP.
  Consulter :ref:`la MIB SNMP <reference-snmp-mib>` pour plus d'information.
* :feature:`429` Ajout d'une variable de substitution ``#TIMESTAMP#`` utilisable
  dans les traitement pré ou post transfert. Cette variable combine les variables
  existantes ``#DATE#`` et ``#HOUR#`` en une seule valeur plus facilement utilisable.
* :feature:`452` Les nouvelles valeurs de substitution ``#BASEFILENAME#`` et
  ``#FILEEXTENSION#`` ont été rajoutées, permettant de récupérer, respectivement
  et séparément, le nom du fichier de transfert et son extension.
* :feature:`456` Ajout d'un paramètre ``output`` à la tâche TRANSFER permettant
  de spécifier le nom/chemin de destination du fichier lorsque celui diffère du
  nom d'origine.
* :feature:`464` Il est désormais possible de configurer la version minimale de
  TLS pour R66-TLS et HTTPS. Cette version minimale peut être renseignée dans
  la configuration protocolaire des client, serveurs et partenaires concernés.
  Pour l'heure, la version minimale par défaut reste toujours la v1.2.
* :feature:`478` Ajout des options ``FilePermissions`` et ``DirectoryPermissions``
  permettant de spécifier les droits attribués au fichiers et dossiers créés par
  la Gateway. Les droits par défauts restent toujours respectivement 0640 pour
  les fichiers et 0750 pour les dossiers.
* :feature:`470` Ajout d'un mécanisme de reprise de transfert automatique en cas
  d'erreur. Pour chaque transfer, il est possible de configurer un nombre d'essais,
  un délai entre chaque essai, et un facteur d'incrément pour ce délai. Il est
  également possible de configurer ces paramètres plus globalement au niveau des
  clients de transfert.
* :feature:`469` Les programmes externes appelés par la tâche EXEC (ou ses variantes)
  héritent désormais des :ref:`valeurs de remplacement<reference-tasks-substitutions>`
  sous forme de variables d'environnement. Il est donc désormais possible de
  référencer ces valeurs dans des programmes externes sans avoir à les fournir via
  les paramètres du programme. Ces variables d'environnement ont exactement le même
  nom que leur valeurs de substitution correspondante (ex: ``#TRUEFULLPATH#``).

* :release:`0.12.9 <2025-07-18>`
* :bug:`482` L'échec du démarrage d'un transfert planifié n'empêche désormais
  plus les autres transferts planifiés de démarrer.
* :bug:`482` Correction d'un bug qui faisait rester les transferts indéfiniment
  en statut *"RUNNING"* sans avancement en cas d'erreur de base de données.
* :bug:`482` Un bug qui empêchait, sous certaines conditions, l'annulation au
  la mise en pause de transferts en cours a été corrigé. Une conséquence de ce
  correctif est que le fonctionnement en grappe **requiert** désormais
  obligatoirement qu'un nom d'instance soit fournis dans le commande de
  lancement (voir :ref:`la documentation<ref-gatewayd-server>` de la commande
  pour plus de détails).
* :feature:`482` Une commande permettant d'exécuter directement des requêtes SQL
  a été ajoutée à l'exécutable serveur ``waarp-gatewayd`` afin de permettre de
  résoudre d'éventuels problèmes de base de données lorsque des outils externes
  ne sont pas disponibles à cette fin.

* :release:`0.12.8 <2025-04-25>`
* :bug:`480` Les clients créés automatiquement lors de l'ajout d'un nouveau
  transfert sont désormais automatiquement démarrés après leur création.
  Précédemment, ces clients n'étaient pas démarrés après création, ce qui les
  rendaient inutilisables sans un redémarrage de l'application.
* :bug:`479` Les droits par défaut des fichiers et dossiers de transfert ont été
  relaxés. Les fichiers reçus ont désormais les droits 640 au lieu de 600. Les
  dossiers créés pour recevoir des fichiers ont eux désormais les droits 750 au
  lieu de 700. À noter que ces changements n'affectent que les systèmes Linux
  (Windows ayant une gestion des droits très différente).

* :release:`0.12.6 <2025-04-25>`
* :bug:`473` Les commandes SNMP prennent désormais les bonnes valeurs pour les
  options SNMPv3 "auth-protocol" et "priv-protocol".
* :bug:`472` Il est désormais possible de "vider" un champ via les commandes
  ``update`` du client terminal. Précédemment, mettre une valeur vide à une
  options laissait le champ inchangé. Désormais, explicitement renseigner une
  valeur vide à une option "effacera" la valeur actuelle du champ en question.
  À noter que omettre l'option entièrement laissera toujours le champ inchangé.

* :release:`0.12.5 <2025-04-18>`
* :bug:`461` La date envoyée dans les notifications SNMP d'erreur de transfert
  est désormais correcte. Précédemment, cette date était systématiquement nulle.
* :bug:`459` Correction d'une fuite de mémoire sur le serveur local R66 et R66-TLS.

* :release:`0.12.4 <2025-04-16>`
* :bug:`455` La tâche *TRANSFER* ne copie plus l'arborescence du chemin source
  en dessous du dossier de règle sur la destination. Cela causait des problèmes
  lorsque le chemin source était absolu. Désormais, le fichier sera toujours
  déposé à la racine du chemin de la règle, et ce, même si le fichier source,
  lui, ne s'y trouvait pas.
* :bug:`457` Les identifiants de pré-connexion PeSIT n'étaient pas correctement
  envoyés par le client de Gateway lorsque celui-ci se connectait à un partenaire.
  Cela est désormais corrigé.

* :release:`0.12.3 <2025-04-03>`
* :bug:`453` Ré-ajout des commandes ``server add`` et ``server delete`` au client
  terminal ``waarp-gateway``. Celles-ci avaient été involontairement retirées en
  version 0.12.1.

* :release:`0.12.2 <2025-04-01>`
* :bug:`-` Correction d'une potentielle situation de concurrence pouvant survenir
  lors de l'exécution parallèle de plusieurs instance d'une même tâche. Les tâches
  concernées par ce problème sont les nouvelles tâches de cryptographie ajoutées
  en version 0.12.0. Cette situation de concurrence pouvait provoquer des erreurs
  imprévues durant l'exécution de ces tâches.
* :bug:`450` Les tâches ``ARCHIVE`` et ``EXTRACT`` qui auraient dû être ajoutée
  en version 0.12.0 ne l'étaient pas. Ces tâches sont maintenant utilisables
  comme prévu.

* :release:`0.12.1 <2025-03-12>`
* :bug:`445` Les clés cryptographiques ne sont désormais plus partagées entre
  plusieurs instances lorsque celles-ci partagent une même base de données
  (excepté dans le cadre d'un fonctionnement en grappe). Les clés sont désormais
  rattachées à une seule instance, et seule celle-ci peut utiliser une clé particulière.
* :bug:`444` Ajout des clés cryptographiques au fichier d'import/export. Celles-ci
  n'avaient pas été ajoutées en version 0.12.0 comme elles auraient dû être.
* :bug:`-` Correction d'une erreur de nommage d'option de la commande terminal
  ``snmp server set``. Le nom court de l'option ``--udp-address`` avait été
  incorrectement défini comme étant ``-u`` au lieu de ``-a``.
* :bug:`444` Ajout de la sous-commande ``key`` au client ligne de commande.
  Celle-ci n'avait pas été ajoutée en 0.12.0 comme elle aurait dû.

* :release:`0.12.0 <2025-03-04>`
* :feature:`-` Mise à jour des pré-requis système. Côté Windows, Waarp Gateway
  requiert désormais au minimum Windows 10 ou Windows Server 2016. Côté Linux,
  un kernel version 3.2 minimum est désormais requis. Toutes les versions
  antérieures de ces OS ne sont désormais plus supportées.
* :feature:`440` Ajout du support pour le protocol PeSIT, ainsi que sa variante
  TLS. À noter que le protocole n'a pas été testé avec d'autres applications, et
  est donc par conséquent en **BETA**. Compte tenu des nombreuses spécificités
  du protocole, il est fortement recommandé de lire :ref:`la rubrique<ref-proto-pesit>`
  spécifiant les détails d'implémentation du protocole avant de commencer à
  l'utiliser.
* :feature:`` Suppression de la contrainte d'unicité sur les identifiants de
  transfert (``remoteTransferID``). Par conséquent, les requêtes de transfert
  entrantes ne seront plus refusées si l'identifiant de transfert fourni par le
  partenaire existe déjà. En revanche, cela signifie qu'il n'est désormais plus
  possible de reprendre un transfert arrêté si un autre transfert avec le même
  identifiant est a été initialisé plus tard. Seul le transfert le plus récent
  avec un identifiant donné peut être repris.
* :feature:`420` Ajout de 2 variables d'environnement ``WAARP_GATEWAYD_CPU_LIMIT``
  et ``WAARP_GATEWAYD_MEMORY_LIMIT`` permettant respectivement de limiter le
  nombre de cœurs CPU ainsi que la mémoire alloués à la Gateway.
* :feature:`-` Ajout de :ref:`handlers REST<rest_keys>` et de :ref:`commandes CLI
  <ref-cli-client-keys>` de gestions des :term:`clés cryptographiques<clé
  cryptographique>` pouvant être utilisées dans les nouvelles tâches cryptographiques.
* :feature:`419` Ajout de plusieurs tâches permettant d'effectuer des tâches
  cryptographiques sur les fichiers de transfert (notamment le chiffrement et la
  signature). Ces tâches sont :

  - ``ENCRYPT`` pour chiffrer un fichier
  - ``DECRYPT`` pour déchiffrer un fichier
  - ``SIGN`` pour signer un fichier
  - ``VERIFY`` pour valider la signature d'un fichier
  - ``ENCRYPT&SIGN`` pour chiffrer et signer un fichier
  - ``DECRYPT&VERIFY`` pour déchiffrer un fichier et valider sa signature

  La documentation complète de ces tâches peut être consultée :ref:`ici<reference-tasks-list>`.
* :feature:`130` Ajout d'une tâche ICAP, permettant (entre autre) d'envoyer
  un fichier de transfert à un service d'analyse antivirus. À noter que cette
  première version de la tâche comporte deux sévères limitations, et est donc
  considérée comme une version *BETA* de la tâche. Voir la :ref:`documentation
  <ref-tasks-icap>` de la tâche pour plus de détails.
* :feature:`65` Ajout des tâches ``ARCHIVE`` et ``EXTRACT`` permettant de créer
  et d'extraire des archives ZIP et TAR, avec possibilité de choisir le type et
  le niveau de compression. Voir la :ref:`documentation des traitements<reference-tasks>`
  pour plus de détails.
* :feature:`63` Ajout de la tâche ``TRANSCODE`` permettant de changer l'encodage
  d'un fichier de transfer. Voir :doc:`la documentation de la tâche TRANSCODE
  <reference/tasks/transcode>` pour plus de détails.

* :release:`0.11.6 <2025-31-01>`
* :bug:`437` Correction du listing de fichier via R66 sous Windows. Précédemment,
  les fichiers renvoyés par le serveur R66 étaient corrects, mais la racine du
  serveur R66 n'était pas correctement retirée des chemins renvoyés (exposant au
  passage l'architecture interne du système de fichiers).
* :bug:`436` Correction d'un crash lors de l'import d'un fichier de configuration
  ne contenant pas de configuration SNMP. La configuration SNMP est désormais
  correctement ignorée lorsqu'elle est absente du fichier d'import.

* :release:`0.11.5 <2025-01-09>`
* :bug:`-` Correction d'un bug dans le *parsing* des chemins sous Windows qui
  empêchait le démarrage de Gateway lorsque les chemins renseignés dans le fichier
  de configuration étaient relatifs.
* :bug:`-` Correction d'un bug de l'API REST qui entravait le bon fonctionnement
  de la commande client ``snmp monitor list``, la faisait systématiquement répondre
  par *"No SNMP monitor found."*. L'API REST renvoie désormais les bonnes informations
  sur les moniteurs SNMP.
* :bug:`433` Ajout d'éléments de configuration manquants du fichier d'import/export.
  Il est donc désormais possible d'importer et exporter :

  - les instances cloud
  - la configuration du serveur SNMP local
  - les moniteurs SNMP distants
  - les autorités d'authentification

* :release:`0.11.4 <2024-17-12>`
* :bug:`-` Lors de l'utilisation des tâches COPY, COPYRENAME, MOVE et MOVERENAME,
  si le dossier de destination n'existe pas, il sera désormais correctement créé.
  Précédemment, un bug empêchait sa création lorsque celui-ci se trouvait sur une
  partition différente du dossier source.
* :bug:`431` Correction d'une régression sur les tâches MOVE et MOVERENAME qui
  empêchait leur bon fonctionnement lorsque la source et la destination se
  trouvaient sur des partitions différentes.

* :release:`0.11.3 <2024-12-11>`
* :bug:`425` Correction d'une mauvaise gestion des erreurs d'initialisation des
  clients de transfert pouvant causer un crash de l'application. La Gateway ne
  devrait désormais plus crasher lorsqu'elle échoue à initialiser un client de
  transfert.
* :bug:`426` Correction d'une erreur d'authentification R66 causé par un bug
  dans l'import des mots de passe R66 via la commande d'import de configuration.

* :release:`0.11.2 <2024-11-27>`
* :bug:`423` Il est désormais possible de mettre à jour les mots de passe serveur
  R66 via la configuration protocolaire (champ "serverPassword"). Précédemment,
  il n'y avait pas de moyen de mettre à jour les mots de passe des serveurs R66
  de cette manière.

* :release:`0.11.1 <2024-11-26>`
* :bug:`421` Correction d'un bug qui empêchait la connection au server R66-TLS
  de la gateway lorsque le client ne présentait pas de certificat et que la
  variable d'environnement ``WAARP_GATEWAY_ALLOW_LEGACY_CERT`` était définie.

* :release:`0.11.0 <2024-09-30>`
* :bug:`413` Correction d'un bug qui entraînait un échec de l'authentification
  des partenaires R66 lorsque leur mot de passe avait été renseigné via la
  configuration protocolaire (champ "serverPassword"). Les mots de passe
  renseignés via la configuration protocolaire R66 devraient dorénavant fonctionner
  correctement.
* :bug:`-` Les paramètres ``"args"`` et ``"delay"`` des diverses tâches *EXEC* -
  spécifiant respectivement les arguments du programme externe, et le temps
  limite d'exécution de la tâche - sont désormais optionnels.
* :bug:`414` Le paramètre ``"using"`` de la tâche *TRANSFER*, spécifiant le
  client à utiliser pour le transfert, est désormais optionnel. Si l'argument
  n'est pas présent, un client par défaut sera utilisé (si possible),
  similairement à si le transfert avait été créé via l'interface REST.
* :bug:`412` Les clients & serveurs locaux ne sont plus automatiquement
  démarrés à leur création via l'interface REST. Un appel au handler ``start``
  est désormais nécessaire pour démarrer les serveurs et clients nouvellement
  créés. À noter cependant que les handlers REST de modification et de suppression
  des serveurs et clients locaux auront toujours pour effet de, respectivement,
  redémarrer et stopper les serveurs et clients concernés.
* :feature:`347` Toutes les réponses aux requêtes faites au serveur HTTP
  d'administration contiennent désormais les informations du serveur (notamment
  sa version) dans l'entête standard "Server". Auparavant, ces informations
  n'était renvoyées que dans les réponses du handler ``/api/about``.
* :feature:`394` Ajout de logging des requêtes REST. Les requêtes faites au
  serveur HTTP d'administration sont désormais loggées au niveau *DEBUG*.
* :feature:`409` Ajout de l'outil de profiling *pprof* au serveur d'administration.
  Cet outil ajoute des handlers au serveur HTTP d'administration qui permettent
  d'exporter divers profils d'activité de l'application. Pour plus de détails,
  consulter la documentation publique de `pprof <https://pkg.go.dev/runtime/pprof>`_
  et de ses `handlers HTTP <https://pkg.go.dev/net/http/pprof>`_.
* :feature:`54` Deuxième partie de l'ajout du service SNMP. Un serveur SNMP a
  a été ajouté permettant de récupérer des informations de diagnostique.
  Consulter :ref:`la MIB SNMP <reference-snmp-mib>` pour plus d'information.
  Ce serveur SNMP peut être configuré via l'API REST et le client terminal.
* :bug:`-` Correction d'une fuite de connexions FTP. Les connexions client FTP
  n'étaient pas correctement fermées, ce qui pouvait conduire à une perte de
  performance, voir même empêcher l'ouverture de nouvelles connexions.
* :feature:`380` Ajout du support pour les instances cloud de type S3. Les fichiers
  de transfert peuvent désormais donc être stockés sur une instance S3. Voir
  la section :ref:`cloud <reference-cloud>` pour avoir plus de détails.
* :feature:`-` Ajout de la commande CLI de gestion des instances cloud.
* :feature:`-` Ajout de la gestion des instances cloud au fichier d'import/export.
* :bug:`-` Ajout des droits d'administration à l'objet ``user`` du fichier
  d'import/export. Les droits d'administration d'un utilisateur étaient
  précédemment perdus lors de l'import ou de l'export de cet utilisateur.

* :release:`0.10.1 <2024-08-29>`
* :bug:`410` Ajout d'une limite à la taille du fichier WAL en cas d'utilisation
  d'une base de données SQLite. Le fichier devrait maintenant être correctement
  tronqué à la fin des transactions. Les connexions à la base de données sont
  également maintenant fermées systématiquement après 2 secondes d'inactivité.
  Cela devrait réduire le risque que des connexions concurrentes empêchent la
  troncature du fichier WAL de s'effectuer en entier.

* :release:`0.10.0 <2024-07-17>`
* :bug:`407` Ajout d'indexes sur les dates de transfert dans les tables
  d'historique. Cela devrait améliorer les performances des requêtes REST et
  des commandes de listing de transferts, en particulier lorsqu'un filtrage
  par date est appliqué.
* :feature:`405` Ajout de la possibilité de filtrer les transferts par ID de
  flux (*followID*) lors du listing de transferts. Ce changement affecte à la
  fois l'API REST et le client terminal, se référer à leur docs respectives
  pour plus de détails.
* :feature:`401` Ajout d'un filtrage d'IP basique permettant de restreindre les
  adresses IP autorisées pour un partenaire cherchant à s'authentifier auprès
  de Gateway. Voir les documentation CLI et REST de gestion des comptes locaux
  pour plus d'information.
* :bug:`406` À la création d'un transfert, si aucun ID de flux (*followID*) n'a
  été spécifié, un ID sera désormais auto-généré. Cet id est visible dans les
  informations de transfert sous le nom ``__followID__``.
* :feature:`54` Première étape de l'ajout d'un service SNMP. La MIB décrivant
  ce service SNMP est disponible :ref:`ici <reference-snmp-mib>`. Pour l'heure,
  celui-ci ne permet que l'envoi de notifications SNMP à un agent tier en cas
  d'erreur de transfert ou en cas d'erreur au démarrage.
  Un serveur SNMP permettant de récupérer des informations de diagnostique sera
  implémenté dans une version ultérieure. Waarp-Gateway supporte SNMPv2 et SNMPv3.

* :release:`0.9.1 <2024-07-01>`
* :bug:`403` Le certificat R66 *legacy* est désormais correctement reconnus
  en tant que tel à sa création, que ce soit via l'import ou via l'API REST.
  Ce certificat n'était pas correctement reconnu depuis la version 0.9.0 quand
  celui-ci était ajouté via l'ancien champ ``certificates``, et sa création
  échouait donc en raison de l'invalidité du certificat.
* :bug:`-` Les mots de passe des compte locaux et des partenaires distants
  peuvent désormais correctement être importés. Un bug introduit en version
  0.9.0 empêchait leur création via le champ ``password`` (pour les comptes
  locaux) ou ``serverPassword`` (pour les partenaires R66).
* :bug:`-` Le cache d'authentification pour mots de passe introduit en version
  0.9.0 fonctionne désormais correctement.
* :bug:`402` L'ancienne propriété "isTLS" des agents R66 (dépréciée en version
  0.7.0 avec la séparation des protocoles R66 et R66-TLS) est de nouveau
  correctement prise en compte. La rétro-compatibilité avec cette propriété
  avait été involontairement rompue avec la mise à jour 0.9.0. Cette
  rétro-compatibilité concerne l'API REST et le fichier d'import/export.

* :release:`0.9.0 <2024-06-05>`
* :feature:`399` Ajout d'un cache d'authentification, permettant d'améliorer
  significativement les performances lorsqu'un grand nombre de demandes de
  transfert sont effectuées en même temps par un même partenaire.
* :bug:`398` Les clé publiques SSH utilisant les algorithmes ``rsa-sha2-256`` et
  ``rsa-sha2-512`` sont désormais correctement acceptées par le client SFTP lors
  de sa connexion à un partenaire. Précédemment, ces algorithmes étaient
  incorrectement refusés par le client SFTP de la gateway malgré le fait qu'ils
  soient supportés.
* :feature:`132` Ajout du support de FTP(S) à la gateway. Il est désormais
  possible d'effectuer des transferts client et serveur avec ce protocole.
  Compte tenu du fonctionnement particulier de ce protocole, il est conseillé de
  lire :ref:`la rubrique<ref-proto-ftp>` spécifiant les détails d'implémentation
  du protocole avant de l'utiliser.
* :bug:`391` Les mots de passe des serveurs locaux R66 sont maintenant bien
  exportés en clair (comme le reste des mots de passe non-hashés).
* :feature:`389` Ajout de le commande ``waarp-gatewayd change-aes-passphrase``
  permettant de changer la passphrase AES utilisée par la *gateway* pour chiffrer
  les mots de passe distants en base de données (voir
  :ref:`la documentation de la commande<reference-cmd-waarp-gatewayd-change-aes>`
  pour plus de détails).
* :feature:`289` Les certificats et les mots de passe sont remplacés par les
  plus génériques "méthodes d'authentification", permettant d'ajouter plus
  facilement de nouvelles formes d'authentification. Pour plus de simplicité,
  l'option *password* des commandes de création des comptes locaux et distants
  est maintenue. Ajout également des "autorités d'authentification" permettant
  de déléguer l'authentification de certains types de partenaires à un tier de
  confiance. Pour plus d'information voir :ref:`le chapitre sur l'authentification
  <reference-auth-methods>`.
* :feature:`-` Ajouter ou enlever des certificats TLS à un agent de transfert
  ne nécessite plus un redémarrage du service en question pour que les
  changements soient pris en compte.
* :feature:`-` Mettre à jour les services (serveurs ou clients) de la gateway
  provoque désormais automatiquement un redémarrage du service en question,
  afin que la nouvelle configuration soit prise en compte. Noter que cela
  interrompra tous les transferts en cours sur le service en question, il est
  donc déconseillé de redémarrer un service si des transferts sont en cours sur
  celui-ci.
* :feature:`-` Les configurations protocolaires client, serveur et partenaire
  sont maintenant séparées les unes des autres, afin qu'elles puissent (lorsque
  cela est nécessaire) avoir des options différentes. Voir
  :ref:`le chapitre sur la configuration protocolaire<reference-proto-config>`
  pour plus de détails.
* :feature:`332` Matérialisation des :term:`clients de transfert<client>`. Les
  clients de transfert de la gateway ne sont dorénavant plus créés à la volé au
  démarrage des transferts, ils doivent désormais avoir été créés au préalable.
  Par conséquent, initialiser un nouveau transfert requiert désormais de préciser
  quel client utiliser pour exécuter ce transfert.
  Par commodité, pour les installations existantes, un client par défaut sera
  créé pour chaque protocole en utilisation lors de la migration de la gateway.
* :bug:`-` Les dossiers par défaut (spécifiés dans le fichier de configuration)
  créés par la gateway ont désormais les permission *740* au lieu de *744*.
* :bug:`-` Dans le cas où la base de données de la gateway est partagée, les
  partenaires de transfert ne sont désormais plus communs à toutes les instances
  utilisant la base. Dans les faits, chaque instance de gateway possède donc
  désormais sont propre annuaire de partenaires, indépendant de ceux des autres
  instances partageant la base de données.

  Lors de la migration de la gateway, pour éviter d'éventuels problème d'incompatibilité,
  tous les partenaires existants ainsi que leurs enfants (comptes distants,
  certificats, etc...) seront dupliqués entre toutes les instances de gateway
  connues utilisant la base de données.
* :feature:`-` Ajout de l'option d'activation/désactivation *disabled* à l'objet
  JSON de serveur local *localAgent* du fichier d'import/export. Il est donc
  désormais possible de spécifier si un serveur importé doit être activé ou
  désactivé.
* :bug:`-` Les nouveaux serveurs locaux créés sont désormais activés par défaut
  au lieu d'être désactivés comme c'était le cas précédemment.

  **Note**: Le terme "activé" ici (*enabled*) ne doit pas être confondu avec
  "actif" (*running*). Les serveurs ne seront pas automatiquement démarré
  immédiatement après leur création. En revanche, ils seront démarrés lors
  du prochain lancement de la gateway.
* :bug:`-` Les *transfer infos* transmises via HTTP(S) sont désormais bien prises
  en compte dans les tâches.
* :bug:`-` Les valeurs de substitution de *transfer info* dans les tâches ne sont
  plus substituées par leur représentation JSON. Cela avait pour effet que les
  valeurs de type *string* étaient substituées avec des guillemets ``"``.
  Désormais, les *transfer info* sont substituées par leur représentation
  textuelle brute.
* :feature:`392` Ajout des argument "copyInfo" et "info" à la tâche `TRANSFER`
  permettant respectivement de copier les *transfer info* du transfer précédent,
  et de définir de nouvelles *transfer info*. Pour plus d'information, voir
  la :ref:`documentation de la tâche TRANSFER<reference-tasks-transfer>`
* :feature:`379` Ajout du support pour les instances cloud en remplacement du
  disque local pour le stockage des fichiers de transfert. Voir la section
  :ref:`cloud <reference-cloud>` pour avoir plus de détails sur l'implémentation
  des différents types d'instances, et la section
  :ref:`gestion des dossiers <gestion_dossiers>` pour plus de détails sur
  leur utilisation.

* :release:`0.8.2 <2024-03-07>`
* :bug:`396` Correction d'une typo dans les mots clés `#TRANSFERID#` et
  `#FULLTRANSFERID#` qui empêchait la substitution de leur valeur de remplacement.

* :release:`0.8.1 <2023-10-23>`
* :bug:`385` Les mots de passes de partenaires R66 importés via la commande
  d'import sont désormais hashés correctement. Depuis la version 0.8.0, les
  partenaires R66 importés via cette commande avaient leurs mots de passe
  hashés incorrectement, ce qui résultait en l'impossibilité pour ces derniers
  de s'authentifier auprès de la *gateway*.
* :bug:`386` Les mots clés de tâche `#ORIGINALFILANAME#` et `#ORIGINALFULLPATH#`
  ont été corrigés pour qu'ils renvoient correctement un nom de fichier.
* :bug:`388` Si l'usage d'une règle est libre, le CLI le montrera désormais
  clairement au lieu d'afficher des listes vides.

* :release:`0.8.0 <2023-06-12>`
* :bug:`376` Correction d'un bug du client R66 de la gateway qui empêchait
  celui-ci récupérer un fichier depuis un agent *Waarp-R66* pour cause de
  "mauvais chemin de fichier".

  Correction également d'un bug de compatibilité avec les agents *Waarp-R66*
  qui pouvait causer un crash de la gateway dans certaines circonstances.
* :feature:`374` Ajout de 2 colonnes ``src_filename`` et ``dest_filename`` aux
  tables des transferts et d'historique. Ces colonnes contiennent respectivement
  (lorsque c'est pertinent) le nom de fichier source, et le nom de fichier
  destination du transfert. Contrairement aux colonnes ``local_path`` et
  ``remote_path`` déjà existante, le contenu de ces 2 nouvelles colonnes ne
  change jamais, même lorsque le nom du fichier est modifié durant le transfert.
  Par conséquent, les nom de fichiers ``src_filename`` et ``dest_filename``
  contiennent toujours le nom de fichier tel qu'il a été donné dans la requête
  originale.

  L'ajout de ces 2 nouvelles colonnes a également permis de corriger 2 bugs
  existants de Gateway:

  1) Les transferts créés avec un chemin de fichier absolus déposaient le fichier
     au mauvais endroit,
  2) Si le nom du fichier changeait durant le transfert, et que le transfert en
     question était ensuite reprogrammé (via la commande ``waarp-gateway transfer retry``),
     le transfert échouait systématiquement avec une erreur "file not found".
* :feature:`375` Il est désormais possible de commencer un transfert d'envoi
  même si le fichier à envoyer n'existe pas encore, tant que celui-ci est créé
  avant le début de la phase d'envoi des données. Typiquement, cela permet de
  démarrer un transfert où le fichier est créé via les pré-tâches.
* :feature:`-` Les logs des tâches (notamment des tâche *exec*) ont été améliorés.
  Dans le cas des tâches exec, la sortie standard du programme externe est
  désormais récupérée et écrite dans les logs de Gateway (au niveau *DEBUG*).
* :bug:`377` Suppression de la limite de temps de 2 secondes imposée par le
  script *updateconf* pour réaliser un import de configuration. Cette limite de
  temps causait l'échec de l'import lorsque celui-ci prenait plus de 2 secondes
  à se compléter.

  Par ailleurs, la commande d'import a été optimisée pour réduire la durée pendant
  laquelle la transaction avec la base de données est active. Cela permet d'éviter
  les conflits entre transactions qui peuvent se produire lorsqu'une transaction
  reste ouverte trop longtemps.

* :release:`0.7.5 <2023-04-07>`
* :bug:`372` Correction d'un bug des tâches ``COPY`` et ``COPYRENAME`` qui
  causait la suppression du contenu du fichier source lorsque celui-ci était
  copié sur lui-même. Dorénavant, copier un fichier sur lui-même n'a plus aucun
  effet.
* :bug:`371` La commande ``rule update`` du client terminal vide correctement
  les chaînes de traitement (pre, post et err) lorsqu'une valeur vide ("") leur
  est attribuée. Précédemment, il n'était pas possible de vider une chaîne de
  traitement existante, attribuer une valeur vide à une chaîne de traitement
  laissait celle-ci inchangée.
* :bug:`370` Ajout de la migration manquante du :ref:`ticket 287<287>` qui faisait
  que tous les serveurs et partenaires R66-TLS créés avant la migration en 0.7.0
  utilisaient R66 en clair au lieu d'utiliser TLS.

* :release:`0.7.4 <2023-03-17>`
* :bug:`367` Les mots clés ``#INPATH#`` et ``#OUTPATH#`` ne concernent que les chemins locaux.
  Les chemins distant peuvent être récupéré à partir du mot clef ``#ORIGINALFULLPATH#``.
* :bug:`365` Correction d'une erreur de la migration 0.7.0 causée par un bug de
  la commande de purge d'historique. Avant la version 0.7.0, la commande de purge
  ne supprimait pas les transfer info liées aux entrées d'historique purgées.
  Par conséquent, il était impossible de migrer vers les version 0.7.X si une
  purge de l'historique avait été effectuée précédemment, et que n'importe
  laquelle des entrées purgée avait des transfer info attachées.
* :bug:`366` Correction d'une erreur empêchant la migration depuis une version
  d'application 0.7.X vers une autre version 0.7.X. La version de la base de
  données n'était pas changée, rendant donc la migration ineffective.

* :release:`0.7.3 <2023-03-06>`
* :bug:`361` Les mots clés ``#INPATH#``, ``#OUTPATH#`` et ``#WORKPATH#`` prennent
  dorénavant bien compte des chemins spécifiés dans les règles et les serveurs
  (précédemment, seuls les dossiers spécifiés dans le fichier de configuration
  étaient pris en compte).

  *Uniquement sous Windows*: Les mots clés ``#TRUEFILENAME#`` et ``#ORIGINALFILENAME#``
  ont été corrigés pour qu'ils renvoient correctement un nom de fichier, comme sous Unix.
* :bug:`363` L'argument "version" de la commande ``waarp-gatewayd migrate`` a
  dorénavant bien une valeur par défaut. Précédemment, omettre cet argument levait
  une erreur. Maintenant, en l'absence de l'argument "version", la commande
  effectuera bien une migration vers la dernière version connue, comme il était
  prévu à l'origine.
* :bug:`362` Correction d'une erreur dans le script de migration de la version
  0.7.0 qui empêchait la migration de s'effectuer à cause de la violation d'une
  contrainte *NOT NULL* sur les tables ``remote_accounts`` et ``crypto_credentials``.

* :release:`0.7.2 <2023-02-15>`
* :bug:`358` Les clients SFTP et R66 ne forcent plus les chemins de fichiers à
  être relatifs. Il est donc désormais possible pour ces clients de requérir
  des chemins absolus et relatifs. Conséquemment, les chemins distants
  (*remote filepath*) calculés lors des transferts peuvent désormais être
  absolus ou relatifs (précédemment, ils étaient forcés à être absolus).

  Á noter que, pour des raisons de sécurité, seuls les clients sont affectés par
  ce changement. Les serveurs de Gateway (quelque soit leur protocole)
  n'acceptent pas les chemins absolus (ces derniers sont considérés comme étant
  relatifs à la racine du serveur).
* :bug:`359` Correction d'un bug du CLI qui causait un crash des commandes
  ``rule list`` et ``rule get`` lorsque la règle à afficher dépassait un certain
  nombre de traitements.

* :release:`0.7.1 <2022-12-19>`
* :bug:`355` Correction de 2 bugs du moteur de migration de base de donnée:

  * Le premier est exclusif aux bases de données SQLite, et causait la suppression
    de tout le contenu des tables enfants lorsque leur table parente était
    modifiée durant une migration (comme c'était le cas pour la version 0.7.0).
  * Le deuxième bug faisait s'exécuter les migrations dans le mauvais ordre lors
    d'un *downgrade* de la base de données, ce qui causait l'échec systématique
    ce celui-ci.
* :bug:`353` Correction d'un bug permettant (lorsque la base de données est partagée)
  à l'interface REST d'une instance de Waarp Gateway de récupérer des entrées
  d'historique ne lui appartenant pas.

* :release:`0.7.0 <2022-12-05>`
* :feature:`351` Ajout des algorithmes suivants à la liste des algorithmes supportés
  par le client et le serveur SFTP de Waarp Gateway:

  - [*Key exchange*] ``diffie-hellman-group-exchange-sha256`` (uniquement côté client)
  - [*Cipher*] ``arcfour256``
  - [*Cipher*] ``arcfour128``
  - [*Cipher*] ``arcfour``
  - [*Cipher*] ``aes128-cbc``
  - [*Cipher*] ``3des-cbc``

  Par ailleurs, tous les algorithmes SSH basés sur SHA-1 sont désormais dépréciés
  (voir la page sur :ref:`la configuration SFTP<proto-config-sftp>` pour la liste
  complète).
* :feature:`276` Ajout d'un *handler* REST et d'une commande terminal
  ``transfer cancel-all`` permettant d'annuler plusieurs transferts d'un coup
  en fonction de leur statut. La documentation de la commande peut être consultée
  :any:`ici <reference/cli/client/transfer/cancel-all>`.
* :feature:`74` Ajout de la commande :ref:`reference-cmd-waarp-gatewayd-restore-history`
  permettant d'importer un dump de l'historique de transfert depuis un fichier JSON.
  Ce dump peut être créé via la nouvelle option ``-e, --export-to`` de la commande
  :ref:`reference-cmd-waarp-gatewayd-purge`.
* :feature:`286` Unifications des *handlers* REST pour les transferts et pour
  l'historique. Tous les transferts (qu'ils soient terminés ou non) sont désormais
  accessibles via le *handler* de transferts. En conséquence, le *handler*
  d'historique est dorénavant déprécié. De même, la commande ``history`` du CLI
  a également été dépréciée, ses fonctions étant désormais assurées par la
  commande ``transfer``.
* :bug:`350` Correction d'une erreur du client R66 causant la réutilisation par
  celui-ci d'anciennes connexions déjà fermées en place et lieu de l'ouverture
  de nouvelles connexions, causant par conséquent l'échec du transfert.
* :feature:`255` Ajout de *handlers* REST permettant l'arrêt et le redémarrage
  des :term:`serveur locaux<serveur>` à chaud. Des sous-commandes ``start``,
  ``stop`` et ``restart`` ont en conséquence été ajoutées à la commande ``server``
  du client en ligne de commande.
* :bug:`346` Correction d'un bug causant l'échec de la validation des chaînes de
  certification comprenant plus de un certificat lors de leur insertion en base
  de données.
* :feature:`187` Ajout d'une commande de purge d'historique à l'exécutable
  ``waarp-gatewayd`` (voir la
  :ref:`documentation de la commande<reference-cmd-waarp-gatewayd-purge>` pour
  plus de détails).
* :feature:`336` Ajout de la possibilité d'activer et désactiver les serveurs
  locaux. Par défaut, les nouveaux serveurs créés sont actifs. Il est désormais
  possible de désactiver un serveur, via :doc:`l'interface REST<reference/cli/client/partner/add>`
  ou via le :doc:`client en ligne de commande<reference/cli/client/server/disable>`.
  Contrairement aux serveurs activés, un serveur désactivé ne sera pas démarré
  automatiquement au lancement de Gateway. À noter que désactiver un serveur
  n'arrête pas immédiatement celui-ci. Le serveur restera actif jusqu'à l'arrêt
  de Gateway ou du serveur en question.
* :feature:`287` _`287` Séparation de R66 et R66-TLS en 2 protocoles distincts. La
  distinction entre les deux se fait désormais via le nom du protocole au lieu
  de la protoConfig. L'option ``isTLS`` de la protoConfig R66 existe toujours
  mais est dorénavant dépréciée.
* :bug:`291` Correction d'une erreur causant l'apparition impromptue de messages
  d'erreur (*warnings*) lorsqu'un client SFTP termine normalement une connexion
  vers un serveur SFTP de Gateway.
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
  d'import. Quand présente, cette option indique à Gateway que la base de
  données doit être vidée avant d'effectuer l'import. Ainsi, tous les éléments
  présents en base concernés par l'opération d'import seront supprimés. Une 2nde
  option nommée ``--force-reset-before-import`` a été ajoutée, permettant aux
  scripts d'outrepasser le message de confirmation de l'option ``-r``.
* :feature:`224` Ajout des utilisateurs Gateway au fichier d'import/export.
  Il est désormais possible d'exporter et importer les utilisateurs Gateway
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
* :bug:`329` Correction de l'impossibilité pour Gateway de se connecter via
  R66-TLS à un agent *Waarp-R66*. Une exception a été ajoutée pour le certificat
  de *Waarp-R66* afin que celui-ci soit accepté par Gateway (voir
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
  Gateway génère dorénavant un identifiant de transfert unique (le
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
  commande su client terminal sans que l'adresse de Gateway soit renseignée.
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
  Gateway au sein d'une grappe d'écraser localement certaines parties de la
  configuration globale de la grappe (voir :ref:`la documentation<reference-conf-override>`
  du fichier d'override de configuration pour plus de détails).
  Pour l'heure, ce fichier permet de définir des remplacement d'adresses pour les
  serveurs locaux, ce qui est nécessaire pour que Gateway fonctionne
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
  d'une Gateway, rendant cette dernière impossible à administrer.
* :bug:`294` Correction d'une erreur dans la réponse des requêtes de listage
  d'utilisateurs sur l'interface REST d'administration (et le client terminal).
  Lorsque la base de données est partagée entre plusieurs *gateways*, l'interface
  d'administration renvoyait indistinctement les utilisateur de toutes les
  *gateways* utilisant cette base de données, au lieu de renvoyer uniquement les
  utilisateurs de l'instance interrogée. Désormais, l'interface REST ne renvoi que
  les utilisateurs de Gateway interrogée. Un problème similaire a également
  été corrigé pour les transferts.
* :feature:`277` Ajout d'une option à la commande `history list` de la CLI
  permettant de trier les entrées de l'historique par date de fin (`stop+` et
  `stop-`). Cette option est également présente sur l'API REST de Gateway.
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
    Pour que cela soit possible, les changements suivants ont été nécessaires :

    - les chemins de règles ne sont plus stockés avec un '/' au début
    - le chemin d'une règle ne peut plus être parent du chemin d'une autre règle
      (par exemple, une règle `/toto/tata` ne peut exister en même temps qu'une
      règle `/toto` car cela créerait des conflits)

* :bug:`-` Les chemins de règle (*path*) ne sont désormais plus stockés avec le
  '/' de début.
* :feature:`247` Ajout d'un client et d'un serveur HTTP/S à Gateway. Il est
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
  Gateway.
* :bug:`272` Correction d'une erreur pouvant survenir lors de l'import d'un
  serveur local dont le nom existe déjà sur une autre instance de Gateway
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
* :bug:`-` Gateway refusera désormais de démarrer si la version de la base
  de données est différente de celle du programme.

* :release:`0.4.0 <2021-07-21>`
* :bug:`259` Correction d'un bug causant une erreur après les pré-tâches d'un
  transfer R66 côté serveur.
* :bug:`260` Correction d'une erreur dans l'import des mots de passe de comptes
  locaux R66.
* :bug:`133` Correction d'une erreur rendant impossible la répartition de charge
  sur plusieurs instances d'une même Gateway. Précédemment, il était possible
  pour 2 instances d'une même Gateway de récupérer un même transfert depuis la
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
  ``WorkDirectory`` du fichier de configuration de Gateway. Ces options ont
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
  disque local de Gateway, le dossier sur l'hôte distant, et le dossier
  temporaire local.
* :feature:`-` Dépréciation des champs JSON ``sourcePath``, ``destPath`` & ``trueFilepath``
  des objets REST de consultation des transferts et de l'historique. Ces champs ont été
  remplacé par les champs ``localPath`` & ``remotePath`` contenant respectivement
  le chemin du fichier sur le disque local de Gateway, et le chemin d'accès au
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
* :bug:`222` Correction d'un comportement incorrect au lancement de Gateway
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
  terminal. Lorsqu'un agent de transfert se connecte à Gateway pour faire
  un transfert, cet identifiant correspond au numéro que cet agent a donné au
  transfert, et qui est donc différent de l'identifiant que Gateway a donné
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
  de l'interface d'administration. Les utilisateurs de Gateway ont désormais
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
* :feature:`129` Ajout d'un client et d'un serveur R66 à Gateway. Il est
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

