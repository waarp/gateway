Modifier un template d'email existant
=====================================

.. http:patch:: /api/email/templates/(string:name)

   Modifie un template d'email existant. Les champs omits resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 201: Le template a été modifié avec succès.
   :statuscode 400: Requête invalide.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: Le template demandé n'existe pas.

   :reqjson string name: Le nom du template.
   :reqjson string subject: Le sujet de l'email.
   :reqjson string mimeType: Le type MIME de l'email. Typiquement soit
     "text/plain" ou "text/html".
   :reqjson string body: Le template du corps de l'email.
   :reqjson array attachments: La liste des fichiers à joindre à l'email.

   :resheader Location: Le chemin d'accès du nouveau template modifié

   **Exemple de requête**

   .. code-block:: http

      PATCH https://my_waarp_gateway.net/api/email/templates/alert_erreur HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 336

      {
        "name": "notif_fin_transfert",
        "subject": "Notification de fin de transfer",
        "mimeType": "text/plain",
        "body": "Notification:

          Le transfert n°#TRANSFERID# du fichier #TRUEFILENAME#
          s'est terminé sans erreur le #DATE# à #HOUR#.

          Waarp"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/email/templates/notif_fin_transfert
