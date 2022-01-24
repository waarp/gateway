Ajouter une règle
=================

.. http:post:: /api/rules

   Ajoute une nouvelle règle à partir des informations renseignées en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de la règle
   :reqjson string comment: Un commentaire optionnel à propos de la règle (description...)
   :reqjson bool isSend: Si vrai, la règle ne peut être utilisée que pour l'envoi
      de fichiers, si faux, la règle ne peut être utilisée que pour la réception
      de fichiers
   :reqjson string path: Le chemin d'identification de la règle. Sert à identifier
      la règle lors d'un transfert si le protocole ne le permet pas. Doit être un
      chemin absolu.
   :reqjson string inPath: Le dossier de destination de la règle. Tous les fichiers
      transférés avec cette règle sont envoyés dans ce dossier. (OBSOLÈTE:
      remplacé par 'localDir' & 'remoteDir')
   :reqjson string outPath: Le dossier source de la règle. Tous les fichiers
      transférés avec cette règle sont récupérés depuis ce dossier. (OBSOLÈTE:
      remplacé par 'localDir' & 'remoteDir')
   :reqjson string workPath: Le dossier temporaire de la règle. Tous les fichiers
      entrants transférés avec cette règle sont déposés dans ce dossier le temps
      du transfert. (OBSOLÈTE: remplacé par 'tmpReceiveDir')
   :reqjson string localDir: Le dossier de la règle sur le disque local de la
      Gateway. Si la règle est une règle d'envoi, ce dossier sert de source au
      fichier; dans le cas contraire, il sert de destination.
   :reqjson string remoteDir: Le chemin de la règle sur l'hôte distant. Si la
      règle est une règle d'envoi, ce dossier sert de destination au fichier;
      dans le cas contraire, il sert de source.
   :reqjson string tmpReceiveDir: Le dossier temporaire local de la règle. Tous
      les fichiers reçu avec cette règle sont déposés dans ce dossier le temps
      du transfert, puis déplacé dans le 'localDir' une fois terminé.
   :reqjson array preTasks: La liste des pré-traitements de la règle.

      * **type** (*string*) - Le type de traitements.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :reqjson array postTasks: La liste des post-traitements de la règle.

      * **type** (*string*) - Le type de traitement.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :reqjson array errorTasks: La liste des traitements d'erreur de la règle.

      * **type** (*string*) - Le type de traitement.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :reqjson object authorized: Les agents autorisés à utiliser cette règle. Par
      défaut, si cet objet est vide, alors la règle peut être utilisée par tous
      le monde, sans exception.

      * **servers** (*array* of *string*) - La liste des serveurs locaux autorisés à utiliser la règle.
      * **partners** (*array* of *string*) - La liste des partenaires distants autorisés à utiliser la règle.
      * **localAccounts** (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.
      * **remoteAccounts** (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.


   :statuscode 201: La règle a été créée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la règle sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès de la nouvelle règle créée


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/rules HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 736

         {
           "name": "règle_1",
           "comment": "ceci est un exemple de règle d'envoi",
           "isSend": true,
           "path": "/chemin/identificateur/de/la/règle",
           "localDir": "/dossier/local",
           "remoteDir": "/dossier/distant",
           "tmpReceiveDir": "/dossier/temporaire",
           "preTasks": [{
             "type": "COPY",
             "args": {"path":"/chemin/de/copie"}
           },{
             "type": "EXEC",
             "args": {"path":"/chemin/du/script","args":"{}","delay":"0"}
           }],
           "postTasks": [{
             "type": "DELETE",
             "args": {}
           },{
             "type": "TRANSFER",
             "args": {"file":"/chemin/du/fichier","to":"waarp_sftp","as":"toto","rule":"règle_2"}
           }],
           "errorTasks": [{
             "type": "MOVE",
             "args": {"path":"/chemin/de/déplacement"}
           },{
             "type": "RENAME",
             "args": {"path":"/chemin/du/renommage"}
           }]
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/rules/règle_1
