Consulter un transfert
======================

.. http:get:: /api/transfers/(int:transfer_id)

   .. deprecated:: 0.4.0

      Les propriétés ``sourcePath`` et ``destPath`` de la réponse ont été
      remplacées par les propriétés ``localFilepath`` et ``remoteFilepath``.

   .. deprecated:: 0.4.0

      La propriété ``trueFilepath`` de la réponse a été remplacée par la
      propriété ``localFilepath``.

   .. deprecated:: 0.5.0

      La propriété ``startDate`` de la réponse a été remplacée par la propriété
      ``start``.

   Renvoie les informations du transfert portant l'identifiant ``transfer_id``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant local du transfert
   :resjson string remoteID: L'identifiant global du transfert
   :resjson bool isServer: Indique si Gateway est agit en tant que serveur
     (``true``) ou en tant que client (``false``)
   :resjson bool isSend: Indique si le transfert est un envoi (``true``) ou une
     réception (``false``)
   :resjson string rule: Le nom de la règle de transfert
   :resjson string requester: Le nom du compte ayant demandé le transfert
   :resjson string requested: Le nom du serveur/partenaire auquel le transfert a été demandé
   :resjson string protocol: Le protocole utilisé pour effectuer le transfert
   :resjson string srcFilename: Le nom du fichier source.
   :resjson string destFilename: Le nom du fichier destination.
   :resjson string localFilepath: Le chemin du fichier sur le disque local
   :resjson string remoteFilepath: Le chemin du fichier sur le partenaire distant
   :resjson number filesize: La taille du fichier (-1 si inconnue)
   :resjson date start: La date de début du transfert
   :resjson date stop: La date de fin du transfert (si le transfert est terminé)
   :resjson string status: Le statut actuel du transfert (valeurs possibles:
     ``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED``, ``ERROR``, ``DONE``
     ou ``CANCELLED``)
   :resjson string step: L'étape actuelle du transfert (valeurs possibles:
     ``StepNone``, ``StepSetup``, ``StepPreTasks``, ``StepData``, ``StepPostTasks``,
     ``StepErrorTasks`` ou ``StepFinalization``)
   :resjson number progress: La progression (en octets) du transfert de données
   :resjson number taskNumber: Le numéro du traitement en cours d'exécution
   :resjson string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjson string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)
   :resjson object transferInfo: Des informations de transfert personnalisées sous
     la forme d'une liste de pairs clé:valeur, c'est-à-dire sous forme d'un objet JSON.
   :resjson number remainingAttempts: Le nombre de tentatives *automatiques* restantes.
     Ce nombre n'inclue donc pas la tentative originale du transfert.
   :resjson string nextAttempt: La date de la prochaine tentative du transfert.
     À noter que ce champ est nul pendant que le transfert est en cours ou si le
     transfers n'a plus de tentatives restantes.
   :resjson number nextRetryDelay: Le délai (en secondes) entre la dernière tentative
     du transfert et la prochaine.
   :resjson number retryIncrementFactor: Le facteur par lequel le délai ci-dessus sera
     multiplié à chaque nouvelle tentative.

   :resjson string trueFilepath: *Déprécié*. Le chemin local complet du fichier 
   :resjson string sourcePath: *Déprécié*. Le fichier source du transfer 
   :resjson string destPath: *Déprécié*. Le fichier destination du transfer 
   :resjson date startDate: *Déprécié*. La date de début du transfert 

   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/transfers/1 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 290

      {
        "id": 1,
        "remoteID": "123456789"
        "rule": "règle_1",
        "isServer": true,
        "isSend": false,
        "requester": "toto",
        "requested": "waarp_sftp",
        "protocol": "sftp",
        "localFilepath": "/chemin/local/fichier1",
        "remoteFilepath": "/chemin/distant/fichier1",
        "filesize": 1234,
        "start": "2019-01-01T02:00:00+02:00",
        "status": "ERROR",
        "step": "DATA",
        "errorCode": "TeDataTransfer",
        "errorMsg": "error during data transfer",
        "progress": 567,
        "transferInfo": {
          "key1": "val1",
          "key2": 2,
          "key3": true
        },
        "remainingTries": 3,
        "nextAttempt": "2019-01-01T03:00:00+02:00",
        "nextRetryDelay": 3600,
        "retryIncrementFactor": 1.5
      }
