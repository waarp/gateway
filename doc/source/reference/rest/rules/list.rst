Lister les règles
=================

.. http:get:: /api/rules

   .. deprecated:: 0.5.0

      Les propriétés ``inPath`` et ``outPath`` de la réponse ont été remplacées
      par les propriétés ``localDir`` et ``remoteDir``.

   .. deprecated:: 0.5.0

      La propriété ``workPath`` de la réponse a été remplacée
      par la propriété ``tmpReceiveDir``.

   Renvoie une liste des règles emplissant les critères donnés en paramètres
   de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les règles seront triées *(défaut: name+)*
   :type sortby: [name+|name-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array rules: La liste des règles demandées
   :resjsonarr string name: Le nom de la règle
   :resjsonarr string comment: Un commentaire optionnel à propos de la règle (description...)
   :resjsonarr bool isSend: Si vrai, la règle ne peut être utilisée que pour l'envoi
      de fichiers, si faux, la règle ne peut être utilisée que pour la réception
      de fichiers
   :resjsonarr string path: Le chemin d'identification de la règle. Sert à identifier
      la règle lors d'un transfert si le protocole ne le permet pas. Doit être un
      chemin absolu.
   :resjson string inPath: *Déprécié*. Le dossier de destination de la règle. Tous les fichiers
      transférés avec cette règle sont envoyés dans ce dossier. 
   :resjson string outPath: *Déprécié*. Le dossier source de la règle. Tous les fichiers
      transférés avec cette règle sont récupérés depuis ce dossier. 
   :resjson string workPath: *Déprécié*. Le dossier temporaire de la règle. Tous les fichiers
      entrants transférés avec cette règle sont déposés dans ce dossier le temps
      du transfert. 
   :resjson string localDir: Le dossier de la règle sur le disque local de la
      Gateway. Si la règle est une règle d'envoi, ce dossier sert de source au
      fichier; dans le cas contraire, il sert de destination.
   :resjson string remoteDir: Le chemin de la règle sur l'hôte distant. Si la
      règle est une règle d'envoi, ce dossier sert de destination au fichier;
      dans le cas contraire, il sert de source.
   :resjson string tmpReceiveDir: Le dossier temporaire local de la règle. Tous
      les fichiers reçu avec cette règle sont déposés dans ce dossier le temps
      du transfert, puis déplacé dans le 'localDir' une fois terminé.
   :resjsonarr array preTasks: La liste des pré-traitements de la règle.

      * ``type`` (*string*) - Le type de traitements.
      * ``reception`` (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjsonarr array postTasks: La liste des post-traitements de la règle.

      * ``type`` (*string*) - Le type de traitement.
      * ``reception`` (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjsonarr array errorTasks: La liste des traitements d'erreur de la règle.

      * ``type`` (*string*) - Le type de traitement.
      * ``reception`` (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjsonarr object authorized: Les agents autorisés à utiliser cette règle. Par
      défaut, si cet objet est vide, alors la règle peut être utilisée par tous
      le monde, sans exception.

      * ``servers`` (*array* of *string*) - La liste des serveurs locaux autorisés à utiliser la règle.
      * ``partners`` (*array* of *string*) - La liste des partenaires distants autorisés à utiliser la règle.
      * ``localAccounts`` (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.
      * ``remoteAccounts`` (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/rules?limit=5 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 1304

      {
        "rules": [{
          "name": "règle_1",
          "comment": "ceci est un exemple de règle d'envoi",
          "isSend": true,
          "path": "/chemin/identificateur/de/la/règle/1",
          "localDir": "/dossier/local/1",
          "remoteDir": "/dossier/distant/1",
          "tmpReceiveDir": "/dossier/temporaire/1",
          "preTasks": [{
            "type": "COPY",
            "args": {"path":"/chemin/de/copie"}
          }],
          "postTasks": [{
            "type": "TRANSFER",
            "args": {"file":"/chemin/du/fichier","to":"waarp_sftp","as":"toto","rule":"règle_2"}
          }],
          "errorTasks": [{
            "type": "MOVE",
            "args": {"path":"/chemin/de/déplacement"}
          }],
          "authorized": {
            "servers": ["serveur_sftp"],
            "partners": ["waarp_r66"],
          }
        },{
          "name": "règle_2",
          "comment": "ceci est un exemple de règle de réception",
          "isSend": false,
          "path": "/chemin/identificateur/de/la/règle/2",
          "localDir": "/dossier/local/2",
          "remoteDir": "/dossier/distant/2",
          "tmpReceiveDir": "/dossier/temporaire/2",
          "preTasks": [{
            "type": "EXEC",
            "args": {"path":"/chemin/du/script","args":"{}","delay":"0"}
          }],
          "postTasks": [{
            "type": "DELETE",
            "args": {}
          }],
          "errorTasks": [{
            "type": "RENAME",
            "args": {"path":"/chemin/du/renommage"}
          }],
          "authorized": {
            "servers": ["serveur_http"],
            "partners": ["waarp_sftp"],
          }
        }]
      }
