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