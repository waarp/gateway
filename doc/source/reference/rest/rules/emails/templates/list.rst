Lister les templates d'email
============================

.. http:get:: /api/email/templates

   Liste les templates d'email.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les règles seront triées *(défaut: name+)*
   :type sortby: [name+|name-]

   :statuscode 200: La liste de template a été renvoyée avec succès.
   :statuscode 400: Requête invalide.
   :statuscode 401: Authentification d'utilisateur invalide.

   :resjsonarr string name: Le nom du template.
   :resjsonarr string subject: Le sujet de l'email.
   :resjsonarr string mimeType: Le type MIME de l'email. Typiquement soit
     "text/plain" ou "text/html".
   :resjsonarr string body: Le template du corps de l'email.
   :resjsonarr array attachments: La liste des fichiers à joindre à l'email.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/email/templates HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 731

      [{
        "name": "alerte_erreur",
        "subject": "Alerte erreur de transfert",
        "mimeType": "text/plain",
        "body": "!! ALERTE !!

          Le transfer n°#TRANSFERID# du fichier #TRUEFILENAME#
          a échoué le #DATE# à #HOUR#
          avec le code #ERRORCODE# et le message \"#ERRORMSG#\".

          Waarp",
        "attachments": ["gateway.log"]
      }, {
        "name": "notif_fin_transfert",
        "subject": "Notification de fin de transfer",
        "mimeType": "text/plain",
        "body": "Notification:

          Le transfer n°#TRANSFERID# du fichier #TRUEFILENAME#
          s'est terminé sans erreur le #DATE# à #HOUR#.

          Waarp"
      }]
