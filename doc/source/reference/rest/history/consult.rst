Consulter une entrée de l'historique
====================================

.. http:get:: /api/history/(int:history_id)

   Renvoie les informations du transfert portant l'identifiant ``history_id``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant local du transfert
   :resjson string remoteID: L'identifiant global du transfert
   :resjson bool isServer: Indique si la *gateway* est agit en tant que serveur
     (``true``) ou en tant que client (``false``)
   :resjson bool isSend: Indique si le transfert est un envoi (``true``) ou une
     réception (``false``)
   :resjson string requester: Le nom du compte ayant demandé le transfert
   :resjson string requested: Le nom du partenaire avec lequel le transfert a été effectué
   :resjson string protocol: Le protocole utilisé pour le transfert
   :resjson string sourceFilename: Le nom du fichier avant le transfert (OBSOLÈTE: remplacé par 'localFilepath' & 'remoteFilepath')
   :resjson string destFilename: Le nom du fichier après le transfert (OBSOLÈTE: remplacé par 'localFilepath' & 'remoteFilepath')
   :resjson string localFilepath: Le chemin du fichier sur le disque local
   :resjson string remoteFilepath: Le chemin d'accès au fichier sur le partenaire distant
   :resjson number filesize: La taille du fichier (-1 si inconnue)
   :resjson string rule: Le nom de la règle de transfert
   :resjson date start: La date de début du transfert
   :resjson date stop: La date de fin du transfert
   :resjson string status: Le statut final du transfert (``CANCELLED`` ou ``DONE``)
   :resjson string step: La dernière étape du transfert (``NONE``, ``PRE TASKS``, ``DATA``, ``POST TASKS``, ``ERROR TASKS`` ou ``FINALIZATION``)
   :resjson number progress: La progression (en octets) du transfert de données
   :resjson number taskNumber: Le numéro du dernier traitement exécuté
   :resjson string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjson string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)


   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/history/1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 176

         {
           "id": 1,
           "isServer": true,
           "isSend": false,
           "rule": "règle_sftp",
           "account": "compte_sftp",
           "remote": "serveur_sftp",
           "protocol": "sftp",
           "localFilepath": "/chemin/local/fichier1",
           "remoteFilepath": "/chemin/distant/fichier1",
           "start": "2019-01-01T01:00:00+02:00",
           "stop": "2019-01-01T02:00:00+02:00",
           "status": "DONE",
         }
