Glossaire
=========

.. glossary::

   utilisateur
   (*user*)
      Terme désignant les identifiants d'un administrateur
      (généralement humain) de la *gateway*. Ces identifiants servent uniquement
      à l'authentification sur l'interface d'administration de la *gateway*. À ne
      pas confondre avec un 'compte' (voir ci-dessous).

   règle
   (*rule*)
      Une règle de transfert est l'ensemble des propriétés communes à tous les
      transferts effectués dans un même contexte. Basiquement, tous les transferts
      utilisant une même règle auront le même dossier source, le même dossier
      destination, et les mêmes traitements.

   serveur
   (*server*)
      Désigne un serveur local de la gateway. Les serveurs locaux font partie
      intégrante de la *gateway*, qui est en fait un regroupement de plusieurs
      serveurs qui tournent en parallèle, chacun traitant un protocole particulier,
      avec chacun une adresse distincte. Lorsqu'un partenaire souhaite initier
      un transfert avec la *gateway*, il doit se connecter puis s'authentifier
      auprès d'un de ces serveurs.

   partenaire
   (*partner*)
      Un partenaire désigne tout agent extérieur avec lequel la *gateway* peut
      initier un transfert. Il s'agit donc d'un serveur distant qui n'appartient
      pas à la *gateway*. Cet agent peut être une autre *gateway*, ou bien un
      serveur tier. Bien que la *gateway* puisse initier un transfert avec les
      partenaires qu'elle connait, cela n'implique en aucun cas que ceux-ci
      peuvent initier un transfert avec la *gateway* (voir 'compte local' ci-dessous).

   client
   (*client*)
      Un client est l'élément de la gateway responsable des transfers avec des
      partenaires distants. Lorsqu'un transfert est créé, celui-ci est ensuite
      récupéré depuis la base de données, puis donné au client approprié pour
      que celui-ci exécute le transfert.

   compte local
   (*local account*)
      Un compte local désigne les identifiants avec lesquels un agent extérieur
      s'authentifie auprès d'un serveur local de la *gateway*. Les comptes locaux
      représentent donc l'ensemble des agents externes qui peuvent initier
      un transfert avec la gateway.

   compte distant
   (*remote account*)
      Un compte distant désigne les identifiants avec lesquels la *gateway*
      s'authentifie auprès d'un partenaire distant.

   certificat
   (*certificate*)
      Un 'certificat' est en fait un regroupement d'un certificat TLS et de ses
      clés publiques et privées. Un certificat peut être attaché à un serveur,
      un partenaire ou bien un compte. Lorsqu'il est attaché à un compte,
      un certificat peut également servir à l'authentification.

   contrôleur
   (*controller*)
      Le contrôleur est le service en charge du lancement des transferts programmés.
      Il s'agit d'un service qui interroge la base de données à intervalles
      réguliers pour récupérer les transferts dont la date de début est arrivée.

   instance cloud
   (*cloud* ou *cloud instance*)
      Une instance cloud est un serveur distant pouvant agir en remplacement
      du disque local, lorsqu'un utilisateur souhaite stocker les fichiers sur
      une machine séparée de celle sur laquelle est installée la gateway.

   infos de transfert
   (*transfer info*)
      Une liste de paires clé-valeur définies par l'utilisateur à la création d'un
      transfert. Ces informations sont donc spécifiques à un transfert donné, et
      peuvent par la suite être utilisées dans traitements (pré/post) définis par
      la règle de transfert. Lors du transfert, ces informations sont communiquées
      en même temps que la requête de transfert. Par conséquent, ces informations
      voyagent toujours dans le sens de la connexion (depuis le client vers le
      serveur) et jamais dans l'autre sens.

      À noter que Waarp-Gateway réserve 2 nom de clé pour les informations de
      transfert : ``__userContent__`` qui sert pour la rétro-compatibilité avec
      Waarp-R66, et ``__followID__`` qui contient l'ID de flux utilisé par
      Waarp-Manager.

   information d'authentification
   (*credential*)
      Une information d'authentification - *parfois aussi désigné comme "identifiant",
      "valeur d'authentification" ou "méthode d'authentification"* - est une valeur
      (ou une paire de valeurs) attachée à un serveur, partenaire, compte local
      ou compte distant, et servant à authentifier cet agent. Les identifiants
      d'un agent regroupent toutes les formes d'authentification pouvant
      authentifier l'agent en question. Cela inclue les mots de passes,
      les certificat TLS, etc...

   autorité d'authentification
   (*authentification authority*)
      Une autorité d'authentification (aussi appelée "autorité de certification"
      ou "autorité de confiance") représente un tiers de confiance auquel la
      gateway fait confiance pour certifier l'identité d'un partenaire souhaitant
      se connecter à la gateway. Ainsi, tout identifiant certifié par une autorité
      d'authentification lui-même considéré comme "de confiance" pour les besoins
      de l'authentification. Pour l'heure, les seules autorités de confiance
      acceptées sont les autorités de certification TLS et SSH.

   moniteur SNMP
   (*SNMP monitor*)
      Un moniteur SNMP est une application tierce apte à recevoir des notifications
      SNMPv2 ou SNMPv3 en provenance de Waarp-Gateway lorsque celle-ci rencontre
      un problème.

   notification SNMP
   (*SNMP trap*)
      Une notification SNMP (ou *trap SNMP*) est un message asynchrone envoyé
      par Waarp-Gateway à un ou plusieurs moniteurs SNMP en cas d'erreur. Depuis
      SNMPv2, il existe deux types de notifications : les *traps* et les *informs*.
      La seule différence étant que les *informs* doivent être acquittés par le
      récepteur, alors que les *traps* ne le sont pas.

   clé cryptographique
   (*cryptographic key*)
      Une clé cryptographique est un nom générique désignant un contenant pour
      des clés servant à effectuer des opérations cryptographiques, telles que
      le (dé)chiffrement ou encore la signature de fichiers. Ces clés sont
      stockées dans une table dédiée en base de données, et peuvent être
      référencées dans les tâches effectuant des opérations cryptographiques.
      Selon le type et l'utilisation de cette clé cryptographique, celle-ci devra
      contenir soit une clé privée, soit une clé publique (voir même parfois les
      deux). Le format de la clé dépend également du type de la clé.

   transfert
   (*transfer*)
      Un transfert est l'opération d'envoi ou de réception d'un fichier entre la
      *gateway* et un partenaire distant. Un transfert est toujours associé à une
      :term:`règle`, un :term:`partenaire`, et un :term:`compte distant` (si la
      gateway est à l'initiative) ou un :term:`compte local` (si le partenaire
      est à l'initiative). Un transfert passe par plusieurs étapes : pré-traitement,
      transfert de données, post-traitement.

   tâche
   traitement
   (*task*)
      Une tâche (ou traitement) est une opération exécutée automatiquement avant
      (pré-traitement), après (post-traitement) ou en cas d'erreur (traitement
      d'erreur) d'un transfert. Les tâches sont définies dans les :term:`règles<règle>`
      et s'exécutent séquentiellement. Exemples : copie, déplacement, chiffrement,
      compression, envoi d'email, exécution de script, acquittement PeSIT.

   condition
   (*condition*)
      Expression optionnelle évaluée avant l'exécution d'une :term:`tâche`. Si la
      condition n'est pas remplie, la tâche est silencieusement ignorée. Les
      conditions supportent les opérateurs de comparaison (``==``, ``!=``, ``>``,
      ``<``), les patterns (``MATCHES``, ``CONTAINS``), les combinaisons logiques
      (``AND``, ``OR``, ``NOT``) et la substitution de variables.

   pré-enregistrement
   (*preregister*)
      Action de créer un transfert avec le statut ``AVAILABLE`` en attente d'une
      requête de pull d'un partenaire. Le transfert pré-enregistré spécifie le
      fichier à mettre à disposition, la règle à appliquer, et le compte autorisé.
      C'est le mécanisme utilisé pour le pull avec pattern.

   Store and Forward
   (*store and forward*)
      Mode de transfert PeSIT dans lequel un fichier transite par un ou plusieurs
      noeuds intermédiaires (relais) avant d'atteindre sa destination finale.
      L'émetteur original et le destinataire final ne sont pas directement
      connectés. L'acquittement de bout en bout se fait via le mécanisme
      F.MESSAGE, en remontant la chaîne de relais.

   F.MESSAGE
      Unité fonctionnelle PeSIT permettant d'envoyer un message (acquittement ou
      texte libre) entre deux partenaires connectés, sans passer par le cycle
      complet de transfert de fichier. Utilisé principalement pour les acquittements
      de bout en bout en mode :term:`Store and Forward`.

   configuration protocolaire
   (*protocol configuration* ou *protoConfig*)
      Ensemble de paramètres spécifiques au protocole de communication (PeSIT,
      R66, SFTP, FTP, etc.) configurés sur un :term:`serveur`, un :term:`client`
      ou un :term:`partenaire`. Ces paramètres contrôlent le comportement du
      protocole : checkpoints, compression, timeouts, modes de compatibilité, etc.

   mode standard
      Mode de fonctionnement PeSIT par défaut de Waarp Gateway. Le chemin du
      fichier distant contient le nom de la :term:`règle` en préfixe. Le serveur
      identifie la règle à appliquer par correspondance de préfixe.

   mode historique
      Mode de compatibilité PeSIT hérité de la convention bancaire SIT. Le champ
      PI 12 (*Filename*) transmet le nom de la :term:`règle` au lieu du chemin de
      fichier, et le nom physique du fichier est transmis via le champ PI 37
      (*FileLabel*). Ce mode est utilisé pour communiquer avec les produits PeSIT
      du marché utilisant cette convention.

   checkpoint
   point de synchronisation
   (*checkpoint*)
      Mécanisme PeSIT de synchronisation pendant un transfert de données.
      L'émetteur envoie périodiquement une demande de confirmation au récepteur
      pour s'assurer que les données ont été correctement reçues. L'intervalle
      entre checkpoints est configuré via PI 7 (en kilo-octets). Les checkpoints
      permettent la reprise de transfert (*restart*) en cas d'interruption.

   restart
   reprise de transfert
   (*restart*)
      Capacité PeSIT à reprendre un transfert interrompu à partir du dernier
      :term:`checkpoint` confirmé, plutôt que de recommencer depuis le début.
      La reprise est négociée lors de la connexion entre le client et le serveur.

   article
      Unité de segmentation des données dans le protocole PeSIT. Un fichier est
      découpé en articles pour le transfert. Les articles peuvent être de taille
      fixe (``fixed``, PI 31 = 0) ou variable (``variable``, PI 31 = 128). La
      taille d'article est spécifiée par PI 32. Un article n'est **pas**
      équivalent à un fichier : les fichiers volumineux sont découpés en plusieurs
      articles.

   NSDU
   (*Network Service Data Unit*)
      Unité de données du service réseau. En PeSIT sur TCP/IP, chaque message
      protocolaire (FPDU) est encapsulé dans un cadre NSDU avec un préfixe de
      2 octets indiquant la taille du message. Le NSDU est le mécanisme standard
      de cadrage pour PeSIT sur TCP.

   FPDU
   (*Functional Protocol Data Unit*)
      Unité de données du protocole fonctionnel PeSIT. Chaque échange protocolaire
      (connexion, sélection, ouverture, transfert de données, etc.) est transporté
      dans une FPDU typée. Les principaux types sont : CONNECT, CREATE, SELECT,
      OPEN, WRITE, READ, DTF (données), CLOSE, DESELECT, RELEASE, MSG (message).

   PI
   (*Protocol Item*)
      Paramètre individuel au sein d'une :term:`FPDU`. Chaque PI a un numéro
      (ex: PI 3 = identifiant client, PI 7 = intervalle de checkpoint, PI 12 =
      nom de fichier, PI 99 = texte libre). Les PI sont encodés en TLV
      (*Type-Length-Value*).

   pré-connexion
   (*pre-connection*)
      Étape optionnelle du protocole PeSIT survenant avant l'établissement de la
      connexion PeSIT proprement dite. Elle permet l'échange d'un identifiant et
      d'un mot de passe en EBCDIC. Cette étape est un héritage du profil SIT
      bancaire et peut être désactivée pour les communications modernes.
